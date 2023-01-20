/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package cue_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

// testdata: input
var (
	input = []byte(`{
	port:   1
	noShut: true
	mtu:    9000
}`)
	invalidInput = []byte(`{port: 1`)

	transform = []byte(`
package foo

#Input: {
	port:   uint16
	noShut: bool
	mtu:    uint16 | *9000
}

#Template: {
	input: #Input

	let _portName = "Ethernet\(input.port)"

	output: devices: {
		"device1": config: {
			Interface: "\(_portName)": {
				Name:        _portName
				Enabled:     input.noShut
				Mtu:         input.mtu
			}
		}
		"device2": config: {
			Interface: "\(_portName)": {
				Name:        _portName
				Enabled:     input.noShut
				Mtu:         input.mtu
			}
		}
	}
}`)
)

func TestNewValueFromBytes(t *testing.T) {
	cctx := cuecontext.New()
	tests := []struct {
		name    string
		given   []byte
		want    string
		wantErr bool
	}{
		{
			"ok",
			input,
			string(input),
			false,
		},
		{
			"err: cue format",
			invalidInput,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := kcue.NewValueFromBytes(cctx, tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, fmt.Sprint(v))
			}
		})
	}
}

func TestNewValueFromJson(t *testing.T) {
	cctx := cuecontext.New()
	tests := []struct {
		name    string
		given   string
		want    string
		wantErr bool
	}{
		{
			"ok",
			`{"foo": "bar"}`,
			`{foo: "bar"}`,
			false,
		},
		{
			"err: invalid json",
			`{"foo": "bar"`,
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kcue.NewValueFromJson(cctx, []byte(tt.given))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				w, err := kcue.NewValueFromBytes(cctx, []byte(tt.want))
				testhelper.ExitOnErr(t, err)
				assert.True(t, w.Equals(got))
			}
		})
	}
}

func TestNewValueWithInstance(t *testing.T) {
	dir := t.TempDir()
	err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
	testhelper.ExitOnErr(t, err)

	tests := []struct {
		name    string
		files   []string
		wantErr bool
	}{
		{
			"ok",
			[]string{"transform.cue"},
			false,
		},
		{
			"err: not exist",
			[]string{"notExist.cue"},
			true,
		},
		{
			"err: no file given",
			[]string{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := kcue.NewValueWithInstance(cuecontext.New(), tt.files, &load.Config{Dir: dir})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestFormatCue(t *testing.T) {
	cctx := cuecontext.New()
	want := cctx.CompileBytes([]byte(`{
	Interface: "Ethernet1": {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`))
	testhelper.ExitOnErr(t, want.Err())

	got, err := kcue.FormatCue(want)
	assert.Nil(t, err)
	assert.True(t, want.Equals(cctx.CompileBytes(got)))
}

func TestNewAstExpr(t *testing.T) {
	given := map[string]any{
		"intVal":   1,
		"floatVal": 1.1,
		"boolVal":  false,
		"strVal":   "foo",
		"nilVal":   nil,
		"listVal":  []any{1, "foo", true},
		"map": map[string]any{
			"intVal":   1,
			"floatVal": 1.0,
			"boolVal":  true,
			"strVal":   "foo",
			"nilVal":   nil,
			"listVal":  []any{1, "foo", true},
		},
	}
	expr := kcue.NewAstExpr(given)
	cctx := cuecontext.New()
	v := cctx.BuildExpr(expr)
	assert.Nil(t, v.Err())

	tests := []struct {
		path string
		want any
	}{
		{"intVal", 1},
		{"floatVal", 1.1},
		{"boolVal", false},
		{"strVal", `"foo"`},
		{"nilVal", "null"},
		{"listVal", `[1, "foo", true]`},
		{"map.intVal", 1},
		{"map.floatVal", 1.0},
		{"map.boolVal", true},
		{"map.strVal", `"foo"`},
		{"map.nilVal", "null"},
		{"map.listVal", `[1, "foo", true]`},
	}
	for _, tt := range tests {
		got := v.LookupPath(cue.ParsePath(tt.path))
		assert.Equal(t, fmt.Sprint(tt.want), fmt.Sprint(got))
	}
}

func TestCueKindOf(t *testing.T) {
	given := []byte(`#Input: {
	strVal:   string
	intVal:   uint16
	boolVal:  bool
	floatVal: float64
	nullVal:  null
}
`)
	cctx := cuecontext.New()
	val, err := kcue.NewValueFromBytes(cctx, given)
	testhelper.ExitOnErr(t, err)

	assert.Equal(t, cue.StructKind, kcue.CueKindOf(val, ""))
	assert.Equal(t, cue.StructKind, kcue.CueKindOf(val, "#Input"))
	assert.Equal(t, cue.StringKind, kcue.CueKindOf(val, "#Input.strVal"))
	assert.Equal(t, cue.IntKind, kcue.CueKindOf(val, "#Input.intVal"))
	assert.Equal(t, cue.BoolKind, kcue.CueKindOf(val, "#Input.boolVal"))
	assert.Equal(t, cue.NumberKind, kcue.CueKindOf(val, "#Input.floatVal"))
	assert.Equal(t, cue.NullKind, kcue.CueKindOf(val, "#Input.nullVal"))
}

func TestCueCommentOf(t *testing.T) {
	given := []byte(`#Input: {
	// kuesta:"key=1"
	key1:   string
	// kuesta:"key=2"
	key2:   uint16
}
`)
	cctx := cuecontext.New()
	val, err := kcue.NewValueFromBytes(cctx, given)
	testhelper.ExitOnErr(t, err)

	t1, err := kcue.CueKuestaTagOf(val, "#Input.key1")
	assert.Equal(t, "key=1", t1)
	assert.Nil(t, err)
	t2, err := kcue.CueKuestaTagOf(val, "#Input.key2")
	assert.Equal(t, "key=2", t2)
	assert.Nil(t, err)
}

func TestGetKuestaTag(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		want    string
		wantErr bool
	}{
		{
			"ok",
			[]byte(`
#Input : {
    // kuesta:"key=1"
    foo: string
}`),
			"key=1",
			false,
		},
		{
			"ok: multiline",
			[]byte(`
#Input : {
    // foo is foo
    // kuesta:"key=1"
    foo: string
}`),
			"key=1",
			false,
		},
		{
			"ok: multiline trailing another comment",
			[]byte(`
#Input : {
    // kuesta:"key=1"
    // foo is foo
    foo: string
}`),
			"key=1",
			false,
		},
		{
			"ok: no comment",
			[]byte(`
#Input : {
    foo: string
}`),
			"",
			false,
		},
		{
			"ok: no tag",
			[]byte(`
#Input : {
    // foo is foo
    foo: string
}`),
			"",
			false,
		},
		{
			"err: multi tag",
			[]byte(`
#Input : {
    // kuesta:"key=1"
    // kuesta:"key=2"
    foo: string
}`),
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cctx := cuecontext.New()
			val, err := kcue.NewValueFromBytes(cctx, tt.given)
			testhelper.ExitOnErr(t, err)

			tag, err := kcue.GetKuestaTag(val.LookupPath(cue.ParsePath("#Input.foo")))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, tag)
				assert.Nil(t, err)
			}
		})
	}
}

func TestStringConverter(t *testing.T) {
	tests := []struct {
		name    string
		kind    cue.Kind
		val     string
		want    any
		wantErr bool
	}{
		{
			"ok: string",
			cue.StringKind,
			"foo",
			"foo",
			false,
		},
		{
			"ok: int",
			cue.IntKind,
			"2",
			2,
			false,
		},
		{
			"ok: float",
			cue.FloatKind,
			"1.0",
			1.0,
			false,
		},
		{
			"ok: float",
			cue.FloatKind,
			"1.1",
			1.1,
			false,
		},
		{
			"ok: number",
			cue.NumberKind,
			"1.0",
			1.0,
			false,
		},
		{
			"ok: number",
			cue.NumberKind,
			"1.1",
			1.1,
			false,
		},
		{
			"err: struct",
			cue.StructKind,
			`{"foo": "bar"}`,
			`{"foo": "bar"}`,
			true,
		},
		{
			"err: list",
			cue.ListKind,
			`["foo", "bar"]`,
			`["foo", "bar"]`,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convert, err := kcue.NewStrConvFunc(tt.kind)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			got, _ := convert(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}
