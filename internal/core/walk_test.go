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
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestCollectPartialDeviceConfig(t *testing.T) {
	dir := t.TempDir()
	dummy := []byte("dummy")
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "foo", "transform.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "input.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "computed", "device1.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "computed", "device2.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "foo", "two", "input.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "foo", "two", "computed", "device1.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "bar", "transform.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "bar", "one", "input.cue"), dummy))
	testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "bar", "one", "computed", "device1.cue"), dummy))

	t.Run("ok", func(t *testing.T) {
		files, err := core.CollectPartialDeviceConfig(dir, "device1")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(files))
		assert.Contains(t, files, filepath.Join(dir, "foo/one/computed/device1.cue"))
		assert.Contains(t, files, filepath.Join(dir, "foo/two/computed/device1.cue"))
		assert.Contains(t, files, filepath.Join(dir, "bar/one/computed/device1.cue"))
	})

	t.Run("ok: not found", func(t *testing.T) {
		files, err := core.CollectPartialDeviceConfig(dir, "device3")
		assert.Nil(t, err)
		assert.Equal(t, 0, len(files))
	})

	t.Run("err: directory not exist", func(t *testing.T) {
		_, err := core.CollectPartialDeviceConfig("notexist", "device1")
		if assert.Error(t, err) {
			var pathError *fs.PathError
			assert.ErrorAs(t, err, &pathError)
		}
	})
}
