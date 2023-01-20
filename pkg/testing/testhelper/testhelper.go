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

package testhelper

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"testing"
)

func ExitOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Error(string(debug.Stack()))
		t.Fatal(err)
	}
}

func Chdir(t *testing.T, path string) {
	t.Helper()
	cd, err := os.Getwd()
	t.Log(cd)
	must(err)
	ExitOnErr(t, os.Chdir(path))
	t.Cleanup(func() {
		ExitOnErr(t, os.Chdir(cd))
	})
}

func Hash(buf []byte) string {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, bytes.NewBuffer(buf)); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
