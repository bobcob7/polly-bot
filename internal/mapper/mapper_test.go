package mapper_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bobcob7/polly-bot/internal/mapper"
	"github.com/go-test/deep"
)

// Supported data types
// - bools
// - bytes
// - ints
// - floats
// - complex
// - strings
// - bytes
// - slices
// - structs
// - pointers to primitive
// - pointers to structs

func newString(s string) *string {
	return &s
}

func ExampleNewDecoder() {
	s := struct {
		Foo string `map:"foo"`
	}{
		Foo: "bar",
	}
	dec := mapper.NewDecoder(os.LookupEnv)
	err := dec.Decode(&s)
	if err != nil {
		fmt.Println("Found error", err)
	}
	fmt.Println(s)
}

func Test_Decoder(t *testing.T) {
	t.Parallel()
	type testSubStruct struct {
		String string `map:"string"`
	}
	type testStruct struct {
		Bool          bool           `map:"bool"`
		Int           int            `map:"int"`
		Int8          int8           `map:"int8"`
		Int16         int16          `map:"int16"`
		Int32         int32          `map:"int32"`
		Int64         int64          `map:"int64"`
		UInt          uint           `map:"uint"`
		UInt8         uint8          `map:"uint8"`
		UInt16        uint16         `map:"uint16"`
		UInt32        uint32         `map:"uint32"`
		UInt64        uint64         `map:"uint64"`
		Float32       float32        `map:"float32"`
		Float64       float64        `map:"float64"`
		Complex64     complex64      `map:"complex64"`
		Complex128    complex128     `map:"complex128"`
		String        string         `map:"string"`
		Bytes         []byte         `map:"bytes"`
		Slice         []string       `map:"slice"`
		Struct        testSubStruct  `map:"struct"`
		PointerString *string        `map:"pointer_string"`
		PointerStruct *testSubStruct `map:"pointer_struct"`
	}
	var got testStruct
	want := testStruct{
		Bool:          true,
		Int:           -1,
		Int8:          -8,
		Int16:         -16,
		Int32:         -32,
		Int64:         -64,
		UInt:          1,
		UInt8:         8,
		UInt16:        16,
		UInt32:        32,
		UInt64:        64,
		Float32:       32.32,
		Float64:       64.64,
		Complex64:     64i,
		Complex128:    128i,
		String:        "string",
		Struct:        testSubStruct{String: "struct_string"},
		Slice:         []string{"1", "2", "3"},
		Bytes:         []byte{48, 49, 50, 51, 52},
		PointerString: newString("string"),
		PointerStruct: &testSubStruct{String: "pointer_struct_string"},
	}
	dec := mapper.NewDecoder(mapper.MapLookup(
		map[string]string{
			"bool":                  "true",
			"int":                   "-1",
			"int8":                  "-8",
			"int16":                 "-16",
			"int32":                 "-32",
			"int64":                 "-64",
			"uint":                  "1",
			"uint8":                 "8",
			"uint16":                "16",
			"uint32":                "32",
			"uint64":                "64",
			"float32":               "32.32",
			"float64":               "64.64",
			"complex64":             "64i",
			"complex128":            "128i",
			"string":                "string",
			"struct_string":         "struct_string",
			"slice_0":               "1",
			"slice_1":               "2",
			"slice_2":               "3",
			"bytes":                 "MDEyMzQ=",
			"pointer_string":        "string",
			"pointer_struct_string": "pointer_struct_string",
		}))
	err := dec.Decode(&got)
	if err != nil {
		t.Error("error decoding", err)
	} else if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}

func Test_Default_Tag(t *testing.T) {
	t.Parallel()
	type testTaglessStruct struct {
		String string
	}
	var got testTaglessStruct
	want := testTaglessStruct{
		String: "string",
	}
	dec := mapper.NewDecoder(
		mapper.MapLookup(map[string]string{
			"String": "string",
		}))
	err := dec.Decode(&got)
	if err != nil {
		t.Error("error decoding", err)
	} else if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}

func Test_Tag_Defaulter(t *testing.T) {
	t.Parallel()
	type testTaglessStruct struct {
		String string
	}
	var got testTaglessStruct
	want := testTaglessStruct{
		String: "string",
	}
	dec := mapper.NewDecoder(mapper.MapLookup(
		map[string]string{
			"STRING": "string",
		}),
		mapper.WithTagDefaulter(strings.ToUpper),
	)
	err := dec.Decode(&got)
	if err != nil {
		t.Error("error decoding", err)
	} else if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}

func Test_Cool_Separator(t *testing.T) {
	t.Parallel()
	type testSubStruct struct {
		String string `map:"string"`
	}
	type testStruct struct {
		Struct testSubStruct `map:"struct"`
	}
	var got testStruct
	want := testStruct{
		Struct: testSubStruct{
			String: "string",
		},
	}
	dec := mapper.NewDecoder(
		mapper.MapLookup(map[string]string{
			"struct string": "string",
		}),
		mapper.WithSeparator(" "),
	)
	err := dec.Decode(&got)
	if err != nil {
		t.Error("error decoding", err)
	} else if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}
