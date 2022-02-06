package chain

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

type nameMapper struct {
	typToName map[reflect.Type]string
	used      map[string]bool
}

func (n *nameMapper) Has(t reflect.Type) bool {
	_, exists := n.typToName[t]
	return exists
}

// Don't ever use these as variable names: keywords + primitive type names var
var disallowed = map[string]bool{
	// keywords
	"break": true, "default": true, "func": true, "interface": true,
	"select": true, "case": true, "defer": true, "go": true, "map": true,
	"struct": true, "chan": true, "else": true, "goto": true, "package": true,
	"switch": true, "const": true, "fallthrough": true, "if": true,
	"range": true, "type": true, "continue": true, "for": true, "import": true,
	"return": true, "var": true,
	// pre-declared identifiers:
	"bool": true, "byte": true, "complex64": true, "complex128": true,
	"error": true, "float32": true, "float64": true, "int": true, "int8": true,
	"int16": true, "int32": true, "int64": true, "rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "true": true, "false": true, "iota": true, "nil": true,
	"append": true, "cap": true, "close": true, "complex": true, "copy": true,
	"delete": true, "imag": true, "len": true, "make": true, "new": true,
	"panic": true, "print": true, "println": true, "real": true,
	"recover": true,
}

func (n *nameMapper) Reserve(names ...string) {
	if n.typToName == nil {
		n.typToName = map[reflect.Type]string{}
		n.used = map[string]bool{}
	}
	for _, name := range names {
		n.used[name] = true
	}
}

func (n *nameMapper) For(t reflect.Type) string {
	if name, exists := n.typToName[t]; exists {
		return name
	}
	if n.typToName == nil {
		n.Reserve()
	}
	for _, name := range n.options(t) {
		if !disallowed[name] && !n.used[name] {
			n.used[name] = true
			n.typToName[t] = name
			return name
		}
	}
	// This should never happen: The final option should be a completely unique
	// name using the full package name and type name.
	panic(fmt.Errorf("Could not come up with a unique variable name for %s.  "+
		"Used names are %v.\nTyp2Name: %q", t, n.used, n.typToName))
}

func extractCaps(s string) string {
	caps := ""
	for i, r := range s {
		if r == '_' {
			return "" // If there are any underscores in the name, give up on extracting caps.
		}
		if !(unicode.IsLetter(r) || unicode.IsNumber(r)) {
			continue
		}
		if i == 0 || unicode.IsUpper(r) || unicode.IsNumber(r) {
			caps += string(r)
		}
	}
	if len(caps) == 0 {
		caps += string(s[0])
	}
	return strings.ToLower(caps)
}

func upperFirstLetter(s string) string {
	for i, r := range s {
		return string(unicode.ToUpper(r)) + s[i+1:]
	}
	return ""
}

func lowerFirstLetter(s string) string {
	for i, r := range s {
		return string(unicode.ToLower(r)) + s[i+1:]
	}
	return ""
}

func ptrPrefix(t reflect.Type) (string, reflect.Type) {
	var s = ""
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		if t.Kind() == reflect.Ptr {
			s = "p" + s
		} else if t.Kind() == reflect.Slice {
			s = "sliceOf" + s
		}
		t = t.Elem()
	}
	return s, t
}
func assemble(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + upperFirstLetter(name)
}

func pkgNamePrefix(pkg string) string {
	if pkg == "" || pkg == "." {
		return ""
	}
	pkg = strings.ToLower(strings.Replace(pkg, ".", "_", -1))
	pkg = strings.Trim(pkg, "_")
	return pkg + "_"
}

func cleanTypeName(name string) string {
	name = strings.Replace(name, "chan ", "chan_", -1)
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Map(func(r rune) rune {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return '_'
		}
		return r
	}, name)
	name = regexp.MustCompile("_{2,}").ReplaceAllLiteralString(name, "_")
	name = strings.Trim(name, "_")
	return name
}

func (n nameMapper) options(t reflect.Type) []string {
	options := wellKnownTypesAndCommonNames[t]
	prefix, t := ptrPrefix(t)
	short_pkg := pkgNamePrefix(filepath.Base(t.PkgPath()))
	full_pkg := pkgNamePrefix(t.PkgPath())

	name := cleanTypeName(t.Name())
	if name == "" {
		name = cleanTypeName(t.String())
		short_pkg = ""
		full_pkg = ""
	}
	if name != "" {
		lname := lowerFirstLetter(name)
		options = append(options, assemble(prefix, lname))
		options = append(options, assemble(short_pkg+prefix, lname))
		options = append(options, assemble(prefix, lname))
		caps := extractCaps(name)
		if caps != "" {
			options = append(options, assemble(prefix, caps))
			options = append(options, assemble(prefix, string(caps[0])))
		}

		options = append(options, assemble(full_pkg+prefix, lname))
	}
	// uniqueVarNameForType should always make something completely unique, but
	// it's a bit verbose.
	// options = append(options, uniqueVarNameForType(t))
	// As a completely paranoid option, this should absolutely positively make
	// a unique var name:
	options = append(options, fmt.Sprintf("__var%d__", len(n.used)))
	return options
}

var wellKnownTypesAndCommonNames = map[reflect.Type][]string{
	reflect.TypeOf((*http.ResponseWriter)(nil)).Elem(): {"rw", "w"},
	reflect.TypeOf((*http.Request)(nil)):               {"req", "r"},
	reflect.TypeOf(""):                                 {"str"},
	reflect.TypeOf(false):                              {"flag"},
	errorType:                                          {"err"},
}
