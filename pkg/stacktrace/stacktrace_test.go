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

package stacktrace_test

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/nttcom/kuesta/pkg/stacktrace"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestShowStackTrace(t *testing.T) {
	t.Run("nested error", func(t *testing.T) {
		err1 := errors.New("foo")
		err2 := errors.Wrap(err1, "bar")
		err3 := errors.Wrap(err2, "baz")

		buf := &bytes.Buffer{}
		stacktrace.Show(buf, err3)

		found := regexp.MustCompile("testing.tRunner").FindAllIndex(buf.Bytes(), -1)
		assert.Equal(t, 1, len(found))
	})

	t.Run("single error", func(t *testing.T) {
		err := errors.New("foo")

		buf := &bytes.Buffer{}
		stacktrace.Show(buf, err)

		found := regexp.MustCompile("testing.tRunner").FindAllIndex(buf.Bytes(), -1)
		assert.Equal(t, 1, len(found))
	})

	t.Run("nil", func(t *testing.T) {
		buf := &bytes.Buffer{}
		stacktrace.Show(buf, nil)
		t.Log(buf)

		assert.Equal(t, 0, len(buf.Bytes()))
	})
}
