// Package chain is a reflection-based dependency-injected chain of functions
// that powers the sandwich middleware framework.
//
// A Func chain represents a sequence of functions to call along with an initial
// input. The parameters to each function are automatically provided from either
// the initial inputs or return values of earlier functions in the sequence.
//
// In contrast to other dependency-injection frameworks, chain does not
// automatically determine how to provide dependencies -- it merely uses the
// most recently-provided value. This enables chains to report errors
// immediately during the chain construction and, if successfully constructed, a
// chain can always be executed.
//
// # HTTP Middleware Example
//
// As a common example, chains in http handling middleware typically start with
// the http.ResponseWriter and *http.Request provided by the http framework:
//
//	base := chain.Func{}.
//	  Arg((*http.ResponseWriter)(nil)).  // declared as an arg when Run
//	  Arg((*http.Request)(nil)).         // declared as an arg when Run
//
// Given the following functions:
//
//	func GetDB() (*UserDB, error) {...}
//	func GetUserFromRequest(db *UserDB, req *http.Request) (*User, error) {...}
//	func SendUserAsJSON(w http.ResponseWriter, u *User) error {...}
//
//	func GetUserID(r *http.Request) (UserID, error) {...}
//	func (db *UserDB) Lookup(UserID) (*User, error) { ... }
//
//	func SendProjectAsJSON(w http.ResponseWriter, p *Project) error {...}
//
// then these chains would work fine:
//
//	base.Then(
//	  GetDB,              // takes no args ✅, provides *UserDB to later funcs
//	  GetUserFromRequest, // takes *UserDB ✅ and *Request ✅, provides *User
//	  SendUserAsJSON,     // takes ResponseWriter ✅ and *User ✅
//	)
//
//	base.Then(
//	  GetDB,            // takes no args ✅, provides *UserDB to later funcs
//	  GetUserID,        // takes *Request ✅, provides UserID
//	  (*UserDB).Lookup, // takes *UserDB ✅ and UserID ✅, provides *User
//	  SendUserAsJSON,   // takes ResponseWriter ✅ and *User ✅
//	)
//
// but these chains would fail:
//
//	base.Then(
//	  GetUserFromRequest, // takes *UserDB ❌ and *Request ✅
//	  GetDB,              // this *UserDB isn't available yet.
//	  SendUserAsJSON,     //
//	)
//
//	base.Then(
//	  GetDB,             // takes no args ✅, provides *UserDB to later funcs
//	  GetUserID,         // takes *Request ✅, provides UserID
//	  (*UserDB).Lookup,  // takes *UserDB ✅ and UserID ✅, provides *User
//	  SendProjectAsJSON, // takes ResponseWriter ✅ and *Project ❌
//	)
//
//	base.Then(
//	  GetUserFromRequest, // takes *UserDB ❌ and *Request ✅
//	  SendUserAsJSON,     //
//	)
package chain

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"text/tabwriter"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// DefaultErrorHandler is called when an error in the chain occurs and no error
// handler has been registered. Warning! The default error handler is not
// checked to verify that it's arguments can be provided. It's STRONGLY
// recommended to keep this as absolutely simple as possible.
var DefaultErrorHandler interface{} = func(err error) { panic(err) }

// Func defines the chain of functions to invoke when Run. Each Func is
// immutable: all operations will return a new Func chain.
type Func struct{ steps []step }

// step is a single value or handler in the middleware stack. Each step has a
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
	tARG stepType = iota
	tVALUE
	tPRE_HANDLER  // PRE handlers are the normal handlers
	tPOST_HANDLER // POST handlers are deferred handlers
	tERROR_HANDLER
)

// Clone this chain and add the extra steps to the clone.
func (c Func) with(steps ...step) Func {
	s := make([]step, 0, len(c.steps)+len(steps))
	s = append(s, c.steps...)
	s = append(s, steps...)
	return Func{s}
}

// Arg indicates that a value with the specified type will be a parameter to Run
// when the Func is invoked. This is typically necessary to start the chain for
// a given middleware framework. Arg should not be exposed to users of sandwich
// since it bypasses the causal checks and risks runtime errors.
func (c Func) Arg(typeOrInterfacePtr interface{}) Func {
	typ := reflect.TypeOf(typeOrInterfacePtr)
	if typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Interface {
		typ = typ.Elem()
	}
	return c.with(step{tARG, reflect.Value{}, typ})
}

// Set an immediate value. This cannot be used to provide an interface, instead
// use SetAs(...) or With(...) with a function that returns the interface.
func (c Func) Set(value interface{}) Func {
	if value == nil {
		panicf("Set(nil) is not allowed -- " +
			"did you mean to use SetAs(val, (*IFace)(nil))?")
	}
	return c.with(step{tVALUE, reflect.ValueOf(value), reflect.TypeOf(value)})
}

// SetAs provides an immediate value as the specified interface type.
func (c Func) SetAs(value, ifacePtr interface{}) Func {
	val := reflect.ValueOf(value)
	typ := reflect.TypeOf(ifacePtr)
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Interface {
		panicf("ifacePtr must be a pointer to an interface for "+
			"SetAs, instead got %s", typ)
	}
	typ = typ.Elem()
	// It's ok to pass in a nil value here if you want the interface to actually
	// be nil.
	if !val.IsValid() {
		val = reflect.Zero(typ)
	}
	if !val.Type().Implements(typ) {
		panicf("%s doesn't implement %s", val.Type(), typ)
	}
	return c.with(step{tVALUE, val, typ})
}

// Compute what types are available from the reserved values, provide values,
// and function return values of the current handler chain. This excludes
// error handlers and deferred handlers.
func (c Func) typesAvailable() map[reflect.Type]bool {
	m := map[reflect.Type]bool{}
	for _, s := range c.steps {
		switch s.typ {
		case tARG:
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

// Then adds one or more handlers to the middleware chain. It may only accept
// args of types that have already been provided.
func (c Func) Then(handlers ...interface{}) Func {
	steps := make([]step, len(handlers))
	available := c.typesAvailable()
	for i, handler := range handlers {
		fn, err := valueOfFunction(handler)
		if err != nil {
			panicf("%s arg of With(...) %v", ordinalize(i+1), err)
		}
		if err := checkCanCall(available, fn); err != nil {
			panicf("%s arg of With(...) %v", ordinalize(i+1), err)
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
// handlers. It may only accept args of types that have already been provided.
func (c Func) OnErr(errorHandler interface{}) Func {
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
// error handlers have been called. Deferred handlers are executed in reverse
// order that they were registered (most recent first). Deferred handlers can
// accept the error type even if it hasn't been explicitly provided yet. If no
// error has occurred, it will be nil.
func (c Func) Defer(handler interface{}) Func {
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

// MustRun will function chain with the provided args and panic if the args
// don't match the expected arg values.
func (c Func) MustRun(argValues ...interface{}) {
	// This will only ever return an error if the arguments to Run don't match.
	// Runtime failures of the functions in the chain are handled by the
	// registered error handlers (or the default error handler which may panic).
	if err := c.Run(argValues...); err != nil {
		panic(err)
	}
}

// Run executes the function chain. All declared args must be provided in the
// order than they were declared. This will return an error only if the
// arguments do not exactly correspond to the declared args. Interface values
// must be passed as pointers to the interface.
//
// Important note: The returned error is NOT related to whether any the calls of
// chain returns an error -- any errors returned by functions in the chain are
// handled by the registered error handlers.
func (c Func) Run(argValues ...interface{}) error {
	data := map[reflect.Type]reflect.Value{}
	postSteps := []step{} // collect post steps here
	errHandler := step{   // Initialize using the default error handler.
		tERROR_HANDLER,
		reflect.ValueOf(DefaultErrorHandler),
		reflect.TypeOf(DefaultErrorHandler),
	}
	stack := []step{}

	// 1: Apply all of the arguments to the available data. Make sure that the
	// provided arguments match the Arg calls, otherwise we bomb.
	if err := c.processRunArgs(data, argValues...); err != nil {
		return err
	}

	// Start executing the function chain. First pass through is the normal call
	// chain, so we skip execution of error handlers and deferred handlers,
	// although we keep track of them.
execution:
	for _, step := range c.steps {
		switch step.typ {
		case tARG:
			// ignored now, already handled during initialization above.
		case tVALUE:
			data[step.val.Type()] = step.val
			data[step.valTyp] = step.val
		case tPRE_HANDLER:
			c.call(step, data, &stack)
			// Check to see if there's an error. If so, abort the chain.
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
	} else {
		data[errorType] = reflect.Zero(errorType)
	}

	// Finally, call any deferred functions that we've gotten to.
	for i := len(postSteps) - 1; i >= 0; i-- {
		c.call(postSteps[i], data, &stack)
	}

	return nil
}

func (c Func) processRunArgs(
	data map[reflect.Type]reflect.Value,
	argValues ...interface{},
) error {
	argIndex := 0
	expectedNumArgs := 0
	var missingArgs []string
	for _, step := range c.steps {
		if step.typ != tARG {
			continue
		}
		expectedNumArgs++
		if argIndex >= len(argValues) {
			missingArgs = append(missingArgs, step.valTyp.String())
			continue
		}
		val := argValues[argIndex]
		argIndex++

		if val == nil {
			if step.valTyp.Kind() == reflect.Interface || step.valTyp.Kind() == reflect.Ptr {
				data[step.valTyp] = reflect.New(step.valTyp).Elem()
				continue
			}
			return fmt.Errorf("bad arg: %s arg of Run(...) should be a %s but is %v",
				ordinalize(argIndex), step.valTyp, val)
		}

		rv := reflect.ValueOf(val)
		if !rv.CanConvert(step.valTyp) {
			return fmt.Errorf("bad arg: %s arg of Run(...) should be a %s but is %s",
				ordinalize(argIndex), step.valTyp, rv.Type())
		}
		data[step.valTyp] = rv.Convert(step.valTyp)
	}
	if len(missingArgs) > 0 {
		return fmt.Errorf("missing args of types: %s", missingArgs)
	}
	if argIndex != len(argValues) {
		return fmt.Errorf("too many args: expected %d args but got %d args",
			expectedNumArgs, len(argValues))
	}
	return nil
}

func (c Func) call(s step, data map[reflect.Type]reflect.Value, stack *[]step) {
	t := s.valTyp
	in := make([]reflect.Value, t.NumIn())
	for i := range in {
		in[i] = data[t.In(i)]
		// This isn't supposed to happen if we've done all our checks right.
		if !in[i].IsValid() {
			name := runtime.FuncForPC(s.val.Pointer()).Name()
			panicf("Cannot inject %s arg of type %s into %s (%s). Data: %v",
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

func (c Func) wrapPanic(x interface{}, steps []step) error {
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

// PanicError is the error that is returned if a handler panics. It includes
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
// noise. The reflect.Value.call removal could affect user stack frames.
//
// TODO(aroman): Refine filtering so that it only removes reflect.Value.call
// frames due to sandwich.
func (p PanicError) FilteredStack() []string {
	lines := strings.Split(p.RawStack, "\n")
	var filtered []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "github.com/augustoroman/sandwich/chain") &&
			!strings.HasPrefix(line, "github.com/augustoroman/sandwich/chain.Func.Run(") &&
			!strings.HasPrefix(line, "github.com/augustoroman/sandwich/chain.Test") {
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
