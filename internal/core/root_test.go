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
	"testing"

	"github.com/nttcom/kuesta/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestRootCfg_Validate(t *testing.T) {
	newValidStruct := func(t func(*core.RootCfg)) *core.RootCfg {
		cfg := &core.RootCfg{
			Verbose:        0,
			Devel:          false,
			ConfigRootPath: "./",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *core.RootCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *core.RootCfg) {},
			false,
		},
		{
			"err: Verbose is over range",
			func(cfg *core.RootCfg) {
				cfg.Verbose = 4
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestRootCfg_MaskCopy(t *testing.T) {
	user := "alice"
	token := "dummy"
	want := "***"
	cfg := &core.RootCfg{
		GitUser:  user,
		GitToken: token,
	}
	cc := cfg.Mask()
	assert.Equal(t, user, cfg.GitUser)
	assert.Equal(t, token, cfg.GitToken)
	assert.Equal(t, user, cc.GitUser)
	assert.Equal(t, want, cc.GitToken)
}
