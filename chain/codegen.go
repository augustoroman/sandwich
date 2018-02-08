package chain

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// Code writes the Go code for the current chain out to w assuming it lives in
// package "pkg" with the specified handler function name
func (c Chain) Code(name, pkg string, w io.Writer) {
	vars := &nameMapper{}

	for _, s := range c.steps {
		vars.Reserve(s.valTyp.Name())
		vars.Reserve(filepath.Base(s.valTyp.PkgPath()))
	}

	fmt.Fprintf(w, "func %s(\n", name)
	for _, s := range c.steps {
		switch s.typ {
		case tVALUE:
			fmt.Fprintf(w, "\t%s %s,\n", vars.For(s.valTyp), strip(pkg, s.valTyp))
		}
	}
	fmt.Fprintf(w, ") func(\n")
	for _, s := range c.steps {
		if s.typ == tRESERVE {
			fmt.Fprintf(w, "\t%s %s,\n", vars.For(s.valTyp), strip(pkg, s.valTyp))
		}
	}
	fmt.Fprintf(w, ") {\n")

	fmt.Fprintf(w, "\treturn func(\n")
	for _, s := range c.steps {
		if s.typ == tRESERVE {
			fmt.Fprintf(w, "\t\t%s %s,\n", vars.For(s.valTyp), strip(pkg, s.valTyp))
		}
	}
	fmt.Fprintf(w, "\t) {\n")

	errHandler := step{tERROR_HANDLER, reflect.ValueOf(DefaultErrorHandler), nil}
	for _, s := range c.steps {
		if s.typ == tRESERVE || s.typ == tVALUE {
			continue
		}

		if s.typ == tERROR_HANDLER {
			errHandler = s
			continue
		}

		for i := 0; i < s.valTyp.NumOut(); i++ {
			t := s.valTyp.Out(i)
			if !vars.Has(t) {
				fmt.Fprintf(w, "\t\tvar %s %s\n", vars.For(t), strip(pkg, t))
			}
		}

		if s.typ == tPOST_HANDLER {
			fmt.Fprintf(w, "\t\tdefer func() {\n\t")
		}

		name, inVars, outVars, returnsError := getArgNames(pkg, vars, s.val)

		fmt.Fprintf(w, "\t\t")
		if len(outVars) > 0 {
			fmt.Fprintf(w, "%s = ", strings.Join(outVars, ", "))
		}
		fmt.Fprintf(w, "%s(%s)\n", name, strings.Join(inVars, ", "))

		if returnsError {
			name, inVars, _, _ := getArgNames(pkg, vars, errHandler.val)
			fmt.Fprintf(w, "\t\tif err != nil {\n")
			fmt.Fprintf(w, "\t\t\t%s(%s)\n", name, strings.Join(inVars, ", "))
			fmt.Fprintf(w, "\t\t\treturn\n")
			fmt.Fprintf(w, "\t\t}\n")
		}

		if s.typ == tPOST_HANDLER {
			fmt.Fprintf(w, "\t\t}()\n")
		}
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "}\n")
}

func strip(pkg string, t reflect.Type) string {
	return stripStr(pkg, t.String())
}
func stripStr(pkg, s string) string {
	pos := strings.IndexFunc(s, func(r rune) bool { return r != '*' })
	s = s[:pos] + strings.TrimPrefix(s[pos:], pkg+".")
	return s
}

func getArgNames(pkg string, vars *nameMapper, v reflect.Value) (name string, in, out []string, returnsError bool) {
	name = runtime.FuncForPC(v.Pointer()).Name()
	name = filepath.Base(name)
	name = strings.TrimPrefix(name, pkg+".")

	if pos := strings.Index(name, ".(*"); pos > 0 {
		pkgName := name[:pos+1]
		pkgName = strings.TrimPrefix(pkgName, pkg+".")
		name = "(*" + pkgName + name[pos+3:]
	}

	t := v.Type()
	out = make([]string, t.NumOut())
	for i := 0; i < t.NumOut(); i++ {
		out[i] = vars.For(t.Out(i))
		if t.Out(i) == errorType {
			returnsError = true
		}
	}
	in = make([]string, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		in[i] = vars.For(t.In(i))
	}
	return name, in, out, returnsError
}

func hasError(str []string) bool {
	for _, s := range str {
		if s == "error" {
			return true
		}
	}
	return false
}
