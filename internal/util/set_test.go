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

package util_test

import (
	"testing"

	"github.com/nttcom/kuesta/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		set := util.NewSet[string]("foo")
		assert.False(t, set.Add("foo"))
		assert.True(t, set.Add("bar"))
		assert.True(t, set.Has("foo"))
		assert.False(t, set.Has("baz"))
		assert.Contains(t, set.List(), "foo")
		assert.Contains(t, set.List(), "bar")
	})

	t.Run("int", func(t *testing.T) {
		set := util.NewSet[int](1)
		assert.False(t, set.Add(1))
		assert.True(t, set.Add(2))
		assert.True(t, set.Has(1))
		assert.False(t, set.Has(3))
		assert.Contains(t, set.List(), 1)
		assert.Contains(t, set.List(), 2)
	})
}
