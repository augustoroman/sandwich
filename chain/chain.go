// Packing chain is a reflection-based dependency-injected handler chain that
// powers the sandwich middleware framework.
package chain

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// DefaultErrorHandler is called when an error in the chain occurs and no error
// handler has been registered.
var DefaultErrorHandler interface{} = func(err error) {
	log.Printf("Unhandled error: %v", err)
}

// Chain holds the stack of middleware handlers to execute.  Chain is immutable:
// all operations will return a new chain.
type Chain struct{ steps []step }

// step is a single value or handler in the middleware stack.  Each step has a
// typ flag that indicates what kind of step it is.
type step struct {
	typ stepType
	val reflect.Value
	// For tVALUE steps, this may optionally be non-nil to specific an
	// additional interface type that is provided.
	// For tRESERVE steps, this must be non-nil to declare the reserved type.
	// For t*_HANDLER steps, this is the function type.
	valTyp reflect.Type
}

type stepType uint8

const (
	tRESERVE stepType = iota
	tVALUE
	tPRE_HANDLER  // PRE handlers are the normal handlers
	tPOST_HANDLER // POST handlers are deferred handlers
	tERROR_HANDLER
)

// Clone this chain and add the extra steps to the clone.
func (c Chain) with(steps ...step) Chain {
	s := make([]step, 0, len(c.steps)+len(steps))
	s = append(s, c.steps...)
	s = append(s, steps...)
	return Chain{s}
}

// Reserve indicates that the specified type will provided by the Run(...) call
// when the chain starts.  This is typically necessary to start the chain for
// a given middleware framework.  Reserve should not be exposed to users of
// sandwich since it bypasses the causal checks and risks runtime errors.
func (c Chain) Reserve(typePtr interface{}) Chain {
	typ := reflect.TypeOf(typePtr)
	if typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Interface {
		typ = typ.Elem()
	}
	return c.with(step{tRESERVE, reflect.Value{}, typ})
}

// Provide an immediate value.  This cannot be used to provide an interface,
// instead use ProvideAs(...) or Then(...) with a function that returns the
// interface.
func (c Chain) Provide(value interface{}) Chain {
	if value == nil {
		panicf("Provide(nil) is not allowed -- " +
			"did you mean to use ProvideAs(val, (*IFace)(nil))?")
	}
	return c.with(step{tVALUE, reflect.ValueOf(value), reflect.TypeOf(value)})
}

// ProvideAs provides an immediate value as the specified interface type.
func (c Chain) ProvideAs(value, ifacePtr interface{}) Chain {
	val := reflect.ValueOf(value)
	typ := reflect.TypeOf(ifacePtr)
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Interface {
		panicf("ifacePtr must be a pointer to an interface for "+
			"ProvideAs, instead got %s", typ)
	}
	typ = typ.Elem()
	if !val.Type().Implements(typ) {
		panicf("%s doesn't implement %s", val.Type(), typ)
	}
	return c.with(step{tVALUE, val, typ})
}

// Compute what types are available from the reserved values, provide values,
// and function return values of the current handler chain.  This excludes
// error handlers and deferred handlers.
func (c Chain) typesAvailable() map[reflect.Type]bool {
	m := map[reflect.Type]bool{}
	for _, s := range c.steps {
		switch s.typ {
		case tRESERVE:
			m[s.valTyp] = true
		case tVALUE:
			m[s.val.Type()] = true
			m[s.valTyp] = true
		case tPRE_HANDLER:
			for i := 0; i < s.valTyp.NumOut(); i++ {
				m[s.valTyp.Out(i)] = true
			}
		case tPOST_HANDLER, tERROR_HANDLER:
			// ignored, we don't allow any return values for these.
		}
	}
	return m
}

// Then adds one or more handlers to the middleware chain.
func (c Chain) Then(handlers ...interface{}) Chain {
	steps := make([]step, len(handlers))
	available := c.typesAvailable()
	for i, handler := range handlers {
		fn, err := valueOfFunction(handler)
		if err != nil {
			panicf("%s arg of Then(...) %v", ordinalize(i+1), err)
		}
		if err := checkCanCall(available, fn); err != nil {
			panicf("%s arg of Then(...) %v", ordinalize(i+1), err)
		}
		fnType := fn.Func.Type()
		steps[i] = step{tPRE_HANDLER, fn.Func, fnType}
		for i := 0; i < fnType.NumOut(); i++ {
			available[fnType.Out(i)] = true
		}
	}
	return c.with(steps...)
}

// OnErr registers an error handler to be called for failures of subsequent
// handlers.
func (c Chain) OnErr(errorHandler interface{}) Chain {
	fn, err := valueOfFunction(errorHandler)
	if err != nil {
		panicf("Error handler %v", err)
	}
	available := c.typesAvailable()
	available[errorType] = true // Set internally by chain.
	if err := checkCanCall(available, fn); err != nil {
		panicf("Error handler %v", err)
	}
	if fn.Func.Type().NumOut() > 0 {
		panicf("Error handler %s may not have any return values, signature is %s",
			fn.Name, fn.Func.Type())
	}
	return c.with(step{tERROR_HANDLER, fn.Func, fn.Func.Type()})
}

// Defer adds a deferred handler to be executed after all normal handlers and
// error handlers have been called.  Deferred handlers are executed in reverse
// order that they were registered (most recent first).
func (c Chain) Defer(handler interface{}) Chain {
	fn, err := valueOfFunction(handler)
	if err != nil {
		panicf("Defer(...) arg %v", err)
	}
	available := c.typesAvailable()
	available[errorType] = true // Set internally by chain.
	if err := checkCanCall(available, fn); err != nil {
		panicf("Defer(...) arg %v", err)
	}
	if fn.Func.Type().NumOut() > 0 {
		panicf("Defer'd handler %s may not have any return values, signature is %s",
			fn.Name, fn.Func.Type())
	}
	return c.with(step{tPOST_HANDLER, fn.Func, fn.Func.Type()})
}

// Run executes the middleware chain using the specified values as initial
// values for all reserved types. All reserved types must be specified, and no
// non-reserved types may be provided here.
func (c Chain) Run(reservedValues ...interface{}) error {
	data := map[reflect.Type]reflect.Value{}
	postSteps := []step{} // collect post steps here
	errHandler := step{tERROR_HANDLER, reflect.ValueOf(DefaultErrorHandler), nil}
	stack := []step{}

	// 1: Apply all of the reserved values.  Make sure that the reserved values
	// match the reserve calls, because otherwise they will be ignored!

	// Apply all of the provided reserved values.
	rvMap := map[reflect.Type]bool{}
	for i, val := range reservedValues {
		if val == nil {
			return fmt.Errorf("reserved value may not be <nil> (%s arg of Run(...))",
				ordinalize(i+1))
		}
		rv := reflect.ValueOf(val)
		rvt := rv.Type()
		if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Interface {
			rv = rv.Elem()
			rvt = rv.Type()
		}
		data[rv.Type()] = rv
		data[rvt] = rv
		rvMap[rvt] = true
	}

	// Ensure that all of the reserved values have been provided
	for _, step := range c.steps {
		if step.typ == tRESERVE {
			if _, provided := data[step.valTyp]; !provided {
				providedReservedValues := make([]string, len(reservedValues))
				for i, val := range reservedValues {
					providedReservedValues[i] = reflect.TypeOf(val).String()
				}
				return fmt.Errorf("Cannot run chain, type %s was reserved "+
					"but no initial value was provided.  Provided types: %s",
					step.valTyp, providedReservedValues)
			}
			delete(rvMap, step.valTyp)
		}
	}

	// Make sure that no non-reserved types have bene provided.
	if len(rvMap) != 0 {
		var extra []string
		for t := range rvMap {
			extra = append(extra, t.String())
		}
		sort.Strings(extra)
		return fmt.Errorf("Run(...) was called with additional types that were "+
			"not reserved: %s", extra)
	}

	// Start executing handlers.  First pass through is the normal call chain,
	// so we skip execution of error handlers and deferred handlers, although
	// we keep track of them.
execution:
	for _, step := range c.steps {
		switch step.typ {
		case tRESERVE:
			// ignored now, already handled during initialization above.
		case tVALUE:
			data[step.val.Type()] = step.val
			data[step.valTyp] = step.val
		case tPRE_HANDLER:
			c.call(step, data, &stack)
			// Check to see if there's an error.  If so, abort the chain.
			if errorVal := data[errorType]; errorVal.IsValid() && !errorVal.IsNil() {
				break execution
			}
		case tPOST_HANDLER:
			postSteps = append(postSteps, step)
		case tERROR_HANDLER:
			errHandler = step
		}
	}

	// Execute the error handler if there is any error.
	if errorVal := data[errorType]; errorVal.IsValid() && !errorVal.IsNil() {
		c.call(errHandler, data, &stack)
	}

	// Finally, call any deferred functions that we've gotten to.
	for _, step := range postSteps {
		c.call(step, data, &stack)
	}

	return nil
}

func (c Chain) call(s step, data map[reflect.Type]reflect.Value, stack *[]step) {
	t := s.valTyp
	in := make([]reflect.Value, t.NumIn())
	for i := range in {
		in[i] = data[t.In(i)]
		// This isn't supposed to happen if we've done all our checks right.
		if !in[i].IsValid() {
			name := runtime.FuncForPC(s.val.Pointer()).Name()
			panicf("Cannot inject %s arg of type %s into %s (%s).  Data: %v",
				ordinalize(i+1), t.In(i), name, t, data)
		}
	}
	defer func() {
		if err := c.wrapPanic(recover(), *stack); err != nil {
			data[errorType] = reflect.ValueOf((*error)(&err)).Elem()
		}
	}()
	*stack = append(*stack, s)
	out := s.val.Call(in)
	for _, val := range out {
		data[val.Type()] = val
	}
}

func (c Chain) wrapPanic(x interface{}, steps []step) error {
	if x == nil {
		return nil
	}
	var stack [8192]byte
	n := runtime.Stack(stack[:], false)

	N := len(steps)
	mwStack := make([]FuncInfo, N)
	for i := range steps {
		step := steps[N-i-1]
		info := runtime.FuncForPC(step.val.Pointer())
		file, line := info.FileLine(step.val.Pointer())
		mwStack[i] = FuncInfo{info.Name(), file, line, step.val}
	}

	return PanicError{
		Val:             x,
		RawStack:        string(stack[:n]),
		MiddlewareStack: mwStack,
	}
}

// PanicError is the error that is returned if a handler panics.  It includes
// the panic'd value (Val), the raw Go stack trace (RawStack), and the
// middleware execution history (MiddlewareStack) that shows what middleware
// functions have already been called.
type PanicError struct {
	Val             interface{}
	RawStack        string
	MiddlewareStack []FuncInfo
}

// FuncInfo describes a registered middleware function.
type FuncInfo struct {
	Name string // fully-qualified name, e.g.: github.com/foo/bar.FuncName
	File string
	Line int
	Func reflect.Value
}

// FilteredStack returns the stack trace without some internal chain.* functions
// and without reflect.Value.call stack frames, since these are generally just
// noise.  The reflect.Value.call removal could affect user stack frames.
//
// TODO(aroman): Refine filtering so that it only removes reflect.Value.call
// frames due to sandwich.
func (p PanicError) FilteredStack() []string {
	lines := strings.Split(p.RawStack, "\n")
	var filtered []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "github.com/augustoroman/sandwich/chain") &&
			!strings.HasPrefix(line, "github.com/augustoroman/sandwich/chain.Chain.Run(") {
			i++
			continue
		}
		if strings.HasPrefix(line, "reflect.Value.call") || strings.HasPrefix(line, "reflect.Value.Call") {
			i++
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func (p PanicError) Error() string {
	var mwStack bytes.Buffer
	w := tabwriter.NewWriter(&mwStack, 5, 7, 2, ' ', 0)
	for _, fn := range p.MiddlewareStack {
		fmt.Fprintf(w, "    %s\t%s\n", fn.Name, fn.Func.Type())
	}
	w.Flush()
	return fmt.Sprintf(
		"Panic executing middleware %s: %v\n"+
			"  Middleware executed:\n%s"+
			"  Filtered call stack:\n    %s",
		p.MiddlewareStack[0].Name, p.Val,
		mwStack.String(),
		strings.Join(p.FilteredStack(), "\n    "))
}
