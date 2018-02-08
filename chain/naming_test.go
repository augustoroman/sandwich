package chain

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameMapper(t *testing.T) {
	var n nameMapper

	assert.Equal(t, "i64", n.For(reflect.TypeOf(int64(0))), "int64")
	assert.Equal(t, "i", n.For(reflect.TypeOf(int(0))), "int")
	assert.Equal(t, "u", n.For(reflect.TypeOf(uint(0))), "uint")
	assert.Equal(t, "f32", n.For(reflect.TypeOf(float32(0))), "float32")
	assert.Equal(t, "f64", n.For(reflect.TypeOf(float64(0))), "float64")
	assert.Equal(t, "flag", n.For(reflect.TypeOf(false)), "bool")

	assert.Equal(t, "rw", n.For(reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()), "http.ResponseWriter")
	assert.Equal(t, "req", n.For(reflect.TypeOf((*http.Request)(nil))), "*http.Request")

	type Req struct{}
	assert.Equal(t, "chain_Req", n.For(reflect.TypeOf(Req{})), "chain.Req (could've been req, but that's taken)")
	assert.Equal(t, "pReq", n.For(reflect.TypeOf(&Req{})), "*chain.Req")
	assert.Equal(t, "pppReq", n.For(reflect.TypeOf((***Req)(nil))), "***chain.Req")

	var c map[string]struct {
		A []byte
		B chan bool
	}
	assert.Equal(t, "map_string_struct_A_uint8_B_chan_bool", n.For(reflect.TypeOf(c)),
		"crazy inlined struct")

	var d map[string]struct {
		A_uint8_B chan bool
	}
	assert.Equal(t, "__var12__", n.For(reflect.TypeOf(d)),
		"inlined struct with var name that conflicts")

	assert.Equal(t, "u8", n.For(reflect.TypeOf(byte(0))), "single byte")
	assert.Equal(t, "sliceOfUint8", n.For(reflect.TypeOf([]byte{})), "byte slice")
	assert.Equal(t, "sliceOfInt", n.For(reflect.TypeOf([]int{})), "int slice")
}
