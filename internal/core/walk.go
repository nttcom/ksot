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

package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nttcom/kuesta/pkg/kuesta"
	errs "github.com/pkg/errors"
)

// CollectPartialDeviceConfig returns list of partial device configs for the given device.
func CollectPartialDeviceConfig(dir, device string) ([]string, error) {
	var files []string
	walkDirFunc := func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return errs.WithStack(fmt.Errorf("walk dir: %w", err))
		}
		if !info.IsDir() {
			return nil
		}
		if info.Name() != kuesta.DirComputed {
			return nil
		}

		p := filepath.Join(path, fmt.Sprintf("%s.cue", device))
		if _, err := os.Stat(p); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return errs.WithStack(fmt.Errorf("check if file exists: %w", err))
		}
		files = append(files, p)
		return filepath.SkipDir
	}

	if err := filepath.WalkDir(dir, walkDirFunc); err != nil {
		return nil, err
	}
	return files, nil
}
