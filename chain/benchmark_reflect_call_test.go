package chain

import (
	"reflect"
	"testing"
)

func twoArg(a, b string) string {
	return a
}

func manyArg(a, b, c, d, e, f, g string) (s, t, u, v, w, y, z string) {
	return "s", "t", "u", "v", "w", "y", "z"
}

func BenchmarkReflectCall_TwoArgs(b *testing.B) {
	arg1, arg2 := reflect.ValueOf("foo"), reflect.ValueOf("bar")
	fn := reflect.ValueOf(twoArg)
	args := []reflect.Value{arg1, arg2}
	for i := 0; i < b.N; i++ {
		fn.Call(args)
	}
}
func BenchmarkDirectCall_TwoArgs(b *testing.B) {
	arg1, arg2 := "foo", "bar"
	for i := 0; i < b.N; i++ {
		twoArg(arg1, arg2)
	}
}

func BenchmarkReflectCall_ManyArgs(b *testing.B) {
	arg := reflect.ValueOf("arg")
	fn := reflect.ValueOf(manyArg)
	args := []reflect.Value{arg, arg, arg, arg, arg, arg, arg}
	for i := 0; i < b.N; i++ {
		fn.Call(args)
	}
}
func BenchmarkDirectCall_ManyArgs(bb *testing.B) {
	a, b, c, d, e, f, g := "a", "b", "c", "d", "e", "f", "g"
	for i := 0; i < bb.N; i++ {
		manyArg(a, b, c, d, e, f, g)
	}
}
