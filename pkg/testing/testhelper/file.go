/*
 Copyright (c) 2023 NTT Communications Corporation

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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// WriteFileWithMkdir writes data to the named file, along with any necessary parent directories.
func WriteFileWithMkdir(path string, buf []byte) error {
	dir, _ := filepath.Split(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0o750); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil { // nolint: gosec
		return errors.WithStack(err)
	}
	return nil
}

// MustGenTgzArchive compresses the file with given content on given path and returns checksum and bytes of the result.
func MustGenTgzArchive(path, content string) (string, io.Reader) {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: path, Mode: 0o600, Size: int64(len(content))}); err != nil {
		panic(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		panic(err)
	}
	must(tw.Close())
	must(gw.Close())

	hasher := sha256.New()
	var out bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(hasher, &out), &buf); err != nil {
		panic(err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	return checksum, &out
}

func MustGenTgzArchiveDir(dir string) (string, io.Reader) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	walkDirFunc := func(path string, info os.FileInfo, err error) error {
		relPath := strings.TrimPrefix(path, dir+string(filepath.Separator))

		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:    relPath,
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}); err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		return nil
	}
	if err := filepath.Walk(dir, walkDirFunc); err != nil {
		panic(err)
	}

	must(tw.Close())
	must(gw.Close())

	hasher := sha256.New()
	var out bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(hasher, &out), &buf); err != nil {
		panic(err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	return checksum, &out
}
