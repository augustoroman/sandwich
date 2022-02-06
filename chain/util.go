package chain

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
)

// TODO(aroman) Replace calls with an explicit error type
func panicf(msgfmt string, args ...interface{}) {
	panic(fmt.Errorf(msgfmt, args...))
}

func valueOfFunction(handler interface{}) (FuncInfo, error) {
	if handler == nil {
		return FuncInfo{}, fmt.Errorf("should be a function, handler is <nil>")
	}
	val := reflect.ValueOf(handler)
	if !val.IsValid() || val.Kind() != reflect.Func {
		return FuncInfo{}, fmt.Errorf("should be a function, handler is %s", val.Type())
	}
	info := runtime.FuncForPC(val.Pointer())
	file, line := info.FileLine(val.Pointer())
	return FuncInfo{info.Name(), file, line, val}, nil
}

func checkCanCall(available map[reflect.Type]bool, fn FuncInfo) error {
	fn_typ := fn.Func.Type()
	for i := 0; i < fn_typ.NumIn(); i++ {
		t := fn_typ.In(i)
		if available[t] {
			continue
		}

		// Un-oh, not available.  Let's see what we can do to make a helpful
		// error message.
		provided := []string{}
		candidates := []string{}
		for typ := range available {
			provided = append(provided, typ.String())
			if t.Kind() == reflect.Interface && typ.Implements(t) {
				candidates = append(candidates, typ.String())
			}
		}
		sort.Strings(provided)

		suggestion := ""
		if len(candidates) == 0 && t.Kind() == reflect.Interface {
			suggestion = fmt.Sprintf(" Type %s is an interface, but not "+
				"implemented by any of the provided types.", t)
		} else if len(candidates) == 1 {
			suggestion = fmt.Sprintf(" Type %s is an interface that is "+
				"implemented by the provided type %s.  Did you mean to use "+
				"'.SetAs(val, (*%s)(nil))' instead of '.Set(val)'?",
				t, candidates[0], strip("main", t))
		} else if len(candidates) > 1 {
			suggestion = fmt.Sprintf(" Type %s is an interface that is implemented "+
				"by %d provided types: %s.  If you meant to use one of those, use "+
				"'.SetAs(val, (*someInterface)(nil))' to explicitly assign "+
				"to that type.",
				t, len(candidates), candidates)
		}

		return fmt.Errorf("can't be called: type %s required for %s arg "+
			"of %s (%s) has not been provided.  Types that have been provided: %s. %s",
			t, ordinalize(i+1), fn.Name, fn_typ, provided, suggestion)
	}
	return nil
}
