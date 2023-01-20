/*
 Copyright (c) 2022-2023 NTT Communications Corporation

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

package kuesta_test

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

// testdata: transform
var (
	input = []byte(`{
	port:   1
	noShut: true
	mtu:    9000
}`)
	missingRequired = []byte(`{
	port:   1
    mtu: 9000
}`)
	missingOptinoal = []byte(`{
	port:   1
	noShut: true
}`)

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

func TestReadServiceMeta(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		want    *kuesta.ServiceMeta
		wantErr bool
	}{
		{
			"ok",
			[]byte(`kind: "foo"`),
			&kuesta.ServiceMeta{
				Kind: "foo",
			},
			false,
		},
		{
			"ok: not found",
			nil,
			nil,
			false,
		},
		{
			"err: invalid format",
			[]byte(`kind: "foo`),
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metadata.yaml")
			if tt.given != nil {
				err := testhelper.WriteFileWithMkdir(path, tt.given)
				testhelper.ExitOnErr(t, err)
			}
			got, err := kuesta.ReadServiceMeta(path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewServiceTransformer(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		wantErr bool
	}{
		{
			"ok",
			transform,
			false,
		},
		{
			"err: invalid cue file",
			[]byte("#Input: {"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), tt.given)
			testhelper.ExitOnErr(t, err)

			cctx := cuecontext.New()
			tr, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, tr.Value())
				assert.Nil(t, tr.Value().Err())
			}
		})
	}
}

func TestServerTransformer_Apply(t *testing.T) {
	dir := t.TempDir()
	err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
	testhelper.ExitOnErr(t, err)

	cctx := cuecontext.New()
	tr, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
	testhelper.ExitOnErr(t, err)

	t.Run("ok", func(t *testing.T) {
		in := cctx.CompileBytes(input)
		testhelper.ExitOnErr(t, in.Err())

		it, err := tr.Apply(in)
		testhelper.ExitOnErr(t, err)

		assert.True(t, it.Next())
		assert.Equal(t, "device1", it.Label())
		assert.True(t, it.Next())
		assert.Equal(t, "device2", it.Label())
		assert.False(t, it.Next())
	})

	t.Run("ok: missing optional fields", func(t *testing.T) {
		in := cctx.CompileBytes(missingOptinoal)
		testhelper.ExitOnErr(t, in.Err())

		_, err := tr.Apply(in)
		assert.Nil(t, err)
	})

	t.Run("err: missing required fields", func(t *testing.T) {
		in := cctx.CompileBytes(missingRequired)
		testhelper.ExitOnErr(t, in.Err())

		_, err := tr.Apply(in)
		assert.Error(t, err)
	})
}

func TestServiceTransformer_ConvertInputType(t *testing.T) {
	transformCue := []byte(`#Input: {
	strVal:   string
	intVal:   uint16
	boolVal:  bool
	floatVal: float64
	nullVal:  null
}`)
	dir := t.TempDir()
	err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transformCue)
	testhelper.ExitOnErr(t, err)

	cctx := cuecontext.New()
	transformer, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
	testhelper.ExitOnErr(t, err)

	tests := []struct {
		name    string
		given   map[string]string
		want    map[string]any
		wantErr bool
	}{
		{
			"ok",
			map[string]string{
				"strVal":   "foo",
				"intVal":   "1",
				"floatVal": "2.0",
				"boolVal":  "true",
				"nullVal":  "anyValue",
			},
			map[string]any{
				"strVal":   "foo",
				"intVal":   1,
				"floatVal": 2.0,
				"boolVal":  true,
				"nullVal":  nil,
			},
			false,
		},
		{
			"err: not exist",
			map[string]string{
				"notExist": "foo",
			},
			nil,
			true,
		},
		{
			"err: cannot convert int",
			map[string]string{
				"intVal": "foo",
			},
			nil,
			true,
		},
		{
			"err: cannot convert float",
			map[string]string{
				"floatVal": "foo",
			},
			nil,
			true,
		},
		{
			"err: cannot convert bool",
			map[string]string{
				"boolVal": "foo",
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.ConvertInputType(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestServiceTransformer_InputKeys(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		want    []string
		wantErr bool
	}{
		{
			"ok",
			[]byte(`
#Input: {
	boolVal:  bool
	// kuesta:"key=1"
	foo:   string
	floatVal: float64
}`),
			[]string{"foo"},
			false,
		},
		{
			"ok: multi keys",
			[]byte(`
#Input: {
	// kuesta:"key=2"
	bar:   uint16
	boolVal:  bool
	// kuesta:"key=1"
	foo:   string
	floatVal: float64
	nullVal:  null

	// kuesta:"key=3"
	baz:   string
}`),
			[]string{"foo", "bar", "baz"},
			false,
		},
		{
			"ok: with other keys",
			[]byte(`
#Input: {
	// kuesta:"key=2"
	bar:   uint16
	boolVal:  bool
	// kuesta:"key=1"
	foo:   string
	// kuesta:"dummy"
	floatVal: float64
	// kuesta:"dummyKey=dummyVal"
	nullVal:  null

	// kuesta:"key=3"
	baz:   string
}`),
			[]string{"foo", "bar", "baz"},
			false,
		},
		{
			"err: not starting from 1",
			[]byte(`
#Input: {
	// kuesta:"key=2"
	foo:   uint16
}`),
			[]string{"foo", "bar", "baz"},
			true,
		},
		{
			"err: not a sequence starting from 1",
			[]byte(`
#Input: {
	// kuesta:"key=1"
	foo:   string
	// kuesta:"key=3"
	bar:   uint16
}`),
			[]string{"foo", "bar", "baz"},
			true,
		},
		{
			"err: not a number",
			[]byte(`
#Input: {
	// kuesta:"key=1"
	foo:   uint16
	// kuesta:"key=bar"
	bar:   string
}`),
			nil,
			true,
		},
		{
			"err: no keys",
			[]byte(`
#Input: {
	foo:   string
	bar:   uint16
}`),
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), tt.given)
			testhelper.ExitOnErr(t, err)

			cctx := cuecontext.New()
			transformer, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
			testhelper.ExitOnErr(t, err)

			got, err := transformer.InputKeys()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func TestNewDeviceFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		wantErr bool
	}{
		{
			"ok",
			[]byte(`config: {
	Interface: Ethernet1: {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`),
			false,
		},
		{
			"err: invalid format",
			[]byte(`config: {`),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cctx := cuecontext.New()
			device, err := kuesta.NewDeviceFromBytes(cctx, tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, device)
			}
		})
	}
}

func TestDevice_Config(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		cctx := cuecontext.New()
		given := []byte(`
config: {
	Interface: Ethernet1: {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`)
		want := cctx.CompileBytes([]byte(`{
	Interface: "Ethernet1": {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`))
		testhelper.ExitOnErr(t, want.Err())

		device, err := kuesta.NewDeviceFromBytes(cctx, given)
		testhelper.ExitOnErr(t, err)
		got, err := device.Config()
		assert.Nil(t, err)
		assert.True(t, want.Equals(cctx.CompileBytes(got)))
	})

	t.Run("err: config missing", func(t *testing.T) {
		cctx := cuecontext.New()
		given := []byte(`something: {foo: "bar"}`)

		device, err := kuesta.NewDeviceFromBytes(cctx, given)
		testhelper.ExitOnErr(t, err)
		got, err := device.Config()
		assert.Nil(t, got)
		assert.Error(t, err)
	})
}
