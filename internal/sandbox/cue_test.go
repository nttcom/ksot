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

package sandbox_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/json"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestCueExtract(t *testing.T) {
	t.Skip()
	jsonVal := []byte(`{"port": 2, "desc": "test"}`)
	cctx := cuecontext.New()
	expr, err := json.Extract("test", jsonVal)
	testhelper.ExitOnErr(t, err)
	v := cctx.BuildExpr(expr)
	t.Fatal(v)
}

func TestCueTypeExtract(t *testing.T) {
	given := []byte(`#Input: {
	device: string
	port:   uint16
	noShut: bool
	desc:   string | *""
	mtu:    uint16 | *9000
}
`)
	cctx := cuecontext.New()
	val, err := kcue.NewValueFromBytes(cctx, given)
	testhelper.ExitOnErr(t, err)

	inputVal := val.LookupPath(cue.ParsePath("#Input"))
	portVal := inputVal.LookupPath(cue.ParsePath("port"))
	deviceVal := inputVal.LookupPath(cue.ParsePath("device"))
	descVal := inputVal.LookupPath(cue.ParsePath("desc"))

	assert.Equal(t, cue.StringKind, deviceVal.IncompleteKind())
	assert.Equal(t, cue.StringKind, descVal.IncompleteKind())
	assert.Equal(t, cue.IntKind, portVal.IncompleteKind())
}
