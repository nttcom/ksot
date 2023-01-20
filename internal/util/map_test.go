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

func TestMergeMap(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4, "d": 5}
	m3 := map[string]int{"b": 6, "c": 7, "e": 8}
	want := map[string]int{"a": 1, "b": 6, "c": 7, "d": 5, "e": 8}
	assert.Equal(t, want, util.MergeMap(m1, m2, m3))
}

func TestSortedMapKeys(t *testing.T) {
	m := map[string]int{"b": 2, "c": 3, "a": 1}
	want := []string{"a", "b", "c"}
	assert.Equal(t, want, util.SortedMapKeys(m))
}
