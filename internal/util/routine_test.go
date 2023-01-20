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
	"context"
	"testing"
	"time"

	"github.com/nttcom/kuesta/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestSetInterval(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		count := 0
		util.SetInterval(context.Background(), func() {
			count++
		}, time.Millisecond)

		assert.Eventually(t, func() bool {
			return count > 2
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("ok: not called after cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		count := 0
		util.SetInterval(ctx, func() {
			count++
			cancel()
		}, time.Millisecond)
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, 1, count)
	})
}
