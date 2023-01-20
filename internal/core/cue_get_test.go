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

package core_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestRunCueGetImpl(t *testing.T) {
	testhelper.Chdir(t, "./testdata")
	called := false
	getter := core.CueGetFunc(func(modPath, outDir string) error {
		assert.Equal(t, "github.com/nttcom/kuesta/testdata", modPath)
		assert.Equal(t, "types/pkg/model", outDir)
		called = true
		return nil
	})
	err := core.RunCueGetImpl(context.Background(), "./pkg/model/sample.go", getter)
	assert.Nil(t, err)

	_, err = os.Stat("./types/pkg/model/sample.go")
	assert.Nil(t, err)
	assert.True(t, called)
}

func TestConvertMapKeyToString(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		want    []byte
		wantErr bool
	}{
		{
			"ok",
			[]byte(`package model

type TestDevice struct {
	Interface map[float]*Interface ` + "`" + `path:"interfaces/interface" module:"openconfig-interfaces/openconfig-interfaces"` + "`" + `
	Vlan      map[uint16]*Vlan      ` + "`" + `path:"vlans/vlan" module:"openconfig-vlan/openconfig-vlan"` + "`" + `
}`),
			[]byte(`package model

type TestDevice struct {
	Interface map[string]*Interface ` + "`" + `path:"interfaces/interface" module:"openconfig-interfaces/openconfig-interfaces"` + "`" + `
	Vlan      map[string]*Vlan      ` + "`" + `path:"vlans/vlan" module:"openconfig-vlan/openconfig-vlan"` + "`" + `
}`),
			false,
		},
		{
			"ok: no change",
			[]byte(`package model

type TestDevice struct {
	Interface map[string]*Interface ` + "`" + `path:"interfaces/interface" module:"openconfig-interfaces/openconfig-interfaces"` + "`" + `
	Vlan      map[string]*Vlan      ` + "`" + `path:"vlans/vlan" module:"openconfig-vlan/openconfig-vlan"` + "`" + `
}`),
			[]byte(`package model

type TestDevice struct {
	Interface map[string]*Interface ` + "`" + `path:"interfaces/interface" module:"openconfig-interfaces/openconfig-interfaces"` + "`" + `
	Vlan      map[string]*Vlan      ` + "`" + `path:"vlans/vlan" module:"openconfig-vlan/openconfig-vlan"` + "`" + `
}`),
			false,
		},
		{
			"err: invalid file",
			[]byte(`package model

type TestDevice struct {`),
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := core.ConvertMapKeyToString("./pkg/model/sample.go", bytes.NewReader(tt.given), buf)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, buf.Bytes())
			}
		})
	}
}
