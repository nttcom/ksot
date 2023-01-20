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

package kuesta_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

func newValidServicePath() *kuesta.ServicePath {
	return &kuesta.ServicePath{
		RootDir: "./tmproot",
		Service: "foo",
		Keys:    []string{"one", "two"},
	}
}

func TestServicePath_Validate(t *testing.T) {
	newValidStruct := func(t func(cfg *kuesta.ServicePath)) *kuesta.ServicePath {
		cfg := newValidServicePath()
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *kuesta.ServicePath)
		wantError bool
	}{
		{
			"ok",
			func(cfg *kuesta.ServicePath) {},
			false,
		},
		{
			"ok: service is empty",
			func(cfg *kuesta.ServicePath) {
				cfg.Service = ""
			},
			false,
		},
		{
			"ok: keys length is 0",
			func(cfg *kuesta.ServicePath) {
				cfg.Keys = nil
			},
			false,
		},
		{
			"err: rootpath is empty",
			func(cfg *kuesta.ServicePath) {
				cfg.RootDir = ""
			},
			true,
		},
		{
			"err: one of keys is empty",
			func(cfg *kuesta.ServicePath) {
				cfg.Keys = []string{"one", ""}
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newValidStruct(tt.transform)
			err := v.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestServicePath_RootPath(t *testing.T) {
	p := newValidServicePath()
	want := filepath.Join("path", "to", "root")
	p.RootDir = "path/to/root"
	assert.Equal(t, want, p.RootPath())
}

func TestServicePath_ServiceDirPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services", p.ServiceDirPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services", p.ServiceDirPath(kuesta.IncludeRoot))
}

func TestServicePath_ServiceItemPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two", p.ServicePath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two", p.ServicePath(kuesta.IncludeRoot))
}

func TestServicePath_ServiceInputPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/input.cue", p.ServiceInputPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/input.cue", p.ServiceInputPath(kuesta.IncludeRoot))
}

func TestServicePath_ReadServiceInput(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), want)
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadServiceInput()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("err: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar", "one", "two"), 0o750)
		testhelper.ExitOnErr(t, err)

		_, err = p.ReadServiceInput()
		assert.Error(t, err)
	})

	t.Run("err: dir not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		p.Keys = []string{"not", "exist"}

		_, err := p.ReadServiceInput()
		assert.Error(t, err)
	})
}

func TestServicePath_WriteServiceInputFile(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	p := newValidServicePath()
	p.RootDir = dir

	err := p.WriteServiceInputFile(buf)
	testhelper.ExitOnErr(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"))
	assert.Nil(t, err)
	assert.Equal(t, buf, got)
}

func TestServicePath_ServiceTransformPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/transform.cue", p.ServiceTransformPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/transform.cue", p.ServiceTransformPath(kuesta.IncludeRoot))
}

func TestServicePath_ReadServiceTransform(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		cctx := cuecontext.New()
		p := newValidServicePath()
		p.RootDir = dir
		want := cctx.BuildExpr(
			ast.NewStruct(
				&ast.Field{Label: ast.NewIdent("test"), Value: ast.NewString("dummy")},
			),
		)
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), []byte(fmt.Sprint(want)))
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadServiceTransform(cctx)
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, fmt.Sprint(want), fmt.Sprint(r.Value()))
		}
	})

	t.Run("err: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar"), 0o750)
		testhelper.ExitOnErr(t, err)

		_, err = p.ReadServiceTransform(cuecontext.New())
		assert.Error(t, err)
	})

	t.Run("err: dir not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		p.Keys = []string{"not", "exist"}

		_, err := p.ReadServiceTransform(cuecontext.New())
		assert.Error(t, err)
	})
}

func TestServicePath_ServiceComputedDirPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/computed", p.ServiceComputedDirPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/computed", p.ServiceComputedDirPath(kuesta.IncludeRoot))
}

func TestServicePath_ServiceComputedPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/computed/device1.cue", p.ServiceComputedPath("device1", kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/computed/device1.cue", p.ServiceComputedPath("device1", kuesta.IncludeRoot))
}

func TestServicePath_ReadServiceComputedFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "computed", "device1.cue"), want)
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadServiceComputedFile("device1")
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("err: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar", "one", "two", "computed"), 0o750)
		testhelper.ExitOnErr(t, err)

		_, err = p.ReadServiceComputedFile("device1")
		assert.Error(t, err)
	})

	t.Run("err: dir not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		p.Keys = []string{"not", "exist"}

		_, err := p.ReadServiceComputedFile("device1")
		assert.Error(t, err)
	})
}

func TestServicePath_WriteServiceComputedFile(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	p := newValidServicePath()
	p.RootDir = dir

	err := p.WriteServiceComputedFile("device1", buf)
	testhelper.ExitOnErr(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "services", "foo", "one", "two", "computed", "device1.cue"))
	assert.Nil(t, err)
	assert.Equal(t, buf, got)
}

func TestServicePath_ServiceMetaPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/metadata.yaml", p.ServiceMetaPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/metadata.yaml", p.ServiceMetaPath(kuesta.IncludeRoot))
}

func TestServicePath_ReadServiceMeta(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := &kuesta.ServiceMeta{
			Kind: "foo",
		}
		given := []byte(`kind: foo`)
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.yaml"), given)
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadServiceMeta()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("ok: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := &kuesta.ServiceMeta{
			Kind: "bar",
		}
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar"), 0o750)
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadServiceMeta()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("err: invalid file format", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		given := []byte(`kind: "foo`)
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.yaml"), given)
		testhelper.ExitOnErr(t, err)

		_, err = p.ReadServiceMeta()
		assert.Error(t, err)
	})
}

func TestServicePath_ReadServiceMetaAll(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		dir := t.TempDir()
		testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.yaml"), []byte(`kind: foo`)))
		testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "services", "bar", "metadata.yaml"), []byte(``)))
		testhelper.ExitOnErr(t, os.MkdirAll(filepath.Join(dir, "services", "baz"), 0o750))

		mlist, err := kuesta.ReadServiceMetaAll(dir)
		assert.Nil(t, err)
		for _, m := range mlist {
			assert.Contains(t, []string{"foo", "bar", "baz"}, m.Kind)
		}
	})

	t.Run("err: service dir not exist", func(t *testing.T) {
		dir := t.TempDir()
		_, err := kuesta.ReadServiceMetaAll(dir)
		assert.Error(t, err)
	})
}

func newValidDevicePath() *kuesta.DevicePath {
	return &kuesta.DevicePath{
		RootDir: "./tmproot",
		Device:  "device1",
	}
}

func TestDevicePath_Validate(t *testing.T) {
	newValidStruct := func(t func(cfg *kuesta.DevicePath)) *kuesta.DevicePath {
		cfg := newValidDevicePath()
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *kuesta.DevicePath)
		wantError bool
	}{
		{
			"ok",
			func(cfg *kuesta.DevicePath) {},
			false,
		},
		{
			"ok: service is empty",
			func(cfg *kuesta.DevicePath) {
				cfg.Device = ""
			},
			false,
		},
		{
			"err: rootpath is empty",
			func(cfg *kuesta.DevicePath) {
				cfg.RootDir = ""
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newValidStruct(tt.transform)
			err := v.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestDevicePath_DeviceDirPath(t *testing.T) {
	p := newValidDevicePath()
	assert.Equal(t, "devices", p.DeviceDirPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/devices", p.DeviceDirPath(kuesta.IncludeRoot))
}

func TestDevicePath_DevicePath(t *testing.T) {
	p := newValidDevicePath()
	assert.Equal(t, "devices/device1", p.DevicePath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/devices/device1", p.DevicePath(kuesta.IncludeRoot))
}

func TestDevicePath_DeviceConfigPath(t *testing.T) {
	p := newValidDevicePath()
	assert.Equal(t, "devices/device1/config.cue", p.DeviceConfigPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/devices/device1/config.cue", p.DeviceConfigPath(kuesta.IncludeRoot))
}

func TestDevicePath_DeviceActualConfigPath(t *testing.T) {
	p := newValidDevicePath()
	assert.Equal(t, "devices/device1/actual_config.cue", p.DeviceActualConfigPath(kuesta.ExcludeRoot))
	assert.Equal(t, "tmproot/devices/device1/actual_config.cue", p.DeviceActualConfigPath(kuesta.IncludeRoot))
}

func TestDevicePath_ReadDeviceConfigFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidDevicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), want)
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadDeviceConfigFile()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("err: file not exist", func(t *testing.T) {
		p := newValidDevicePath()
		p.RootDir = dir
		p.Device = "device2"
		err := os.MkdirAll(filepath.Join(dir, "devices", "device2"), 0o750)
		testhelper.ExitOnErr(t, err)

		_, err = p.ReadDeviceConfigFile()
		assert.Error(t, err)
	})

	t.Run("err: dir not exist", func(t *testing.T) {
		p := newValidDevicePath()
		p.Device = "notExist"
		p.RootDir = dir

		_, err := p.ReadDeviceConfigFile()
		assert.Error(t, err)
	})
}

func TestDevicePath_WriteDeviceConfigFile(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	p := newValidDevicePath()
	p.RootDir = dir

	err := p.WriteDeviceConfigFile(buf)
	testhelper.ExitOnErr(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "devices", "device1", "config.cue"))
	assert.Nil(t, err)
	assert.Equal(t, buf, got)
}

func TestDevicePath_CheckSum(t *testing.T) {
	config := []byte("foobar")

	t.Run("ok", func(t *testing.T) {
		dir := t.TempDir()
		testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config))

		hasher := sha256.New()
		hasher.Write(config)
		want := fmt.Sprintf("%x", hasher.Sum(nil))

		dp := kuesta.DevicePath{RootDir: dir, Device: "device1"}
		got, err := dp.CheckSum()
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("err: config not found", func(t *testing.T) {
		dir := t.TempDir()
		testhelper.ExitOnErr(t, os.MkdirAll(filepath.Join(dir, "devices"), 0o755))

		dp := kuesta.DevicePath{RootDir: dir, Device: "device1"}
		_, err := dp.CheckSum()
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestDevicePath_ReadActualDeviceConfigFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidDevicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "actual_config.cue"), want)
		testhelper.ExitOnErr(t, err)

		r, err := p.ReadActualDeviceConfigFile()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("err: file not exist", func(t *testing.T) {
		p := newValidDevicePath()
		p.RootDir = dir
		p.Device = "device2"
		err := os.MkdirAll(filepath.Join(dir, "devices", "device2"), 0o750)
		testhelper.ExitOnErr(t, err)

		_, err = p.ReadActualDeviceConfigFile()
		assert.Error(t, err)
	})

	t.Run("err: dir not exist", func(t *testing.T) {
		p := newValidDevicePath()
		p.Device = "notExist"
		p.RootDir = dir

		_, err := p.ReadActualDeviceConfigFile()
		assert.Error(t, err)
	})
}

func TestParseServiceInputPath(t *testing.T) {
	tests := []struct {
		name     string
		given    string
		wantSvc  string
		wantKeys []string
		wantErr  bool
	}{
		{
			"ok",
			"services/foo/one/input.cue",
			"foo",
			[]string{"one"},
			false,
		},
		{
			"ok",
			"services/foo/one/two/three/four/input.cue",
			"foo",
			[]string{"one", "two", "three", "four"},
			false,
		},
		{
			"err: not start from services",
			"devices/device1/config.cue",
			"",
			[]string{},
			true,
		},
		{
			"err: file is not input.cue",
			"services/foo/one/computed/device1.cue",
			"",
			[]string{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSvc, gotKeys, err := kuesta.ParseServiceInputPath(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.wantSvc, gotSvc)
				assert.Equal(t, tt.wantKeys, gotKeys)
				assert.Nil(t, err)
			}
		})
	}
}

func TestParseServiceComputedFilePath(t *testing.T) {
	tests := []struct {
		name    string
		given   string
		want    string
		wantErr bool
	}{
		{
			"ok",
			"services/foo/one/computed/device1.cue",
			"device1",
			false,
		},
		{
			"ok",
			"services/foo/one/two/three/four/computed/device2.cue",
			"device2",
			false,
		},
		{
			"err: not start from services",
			"devices/device1/config.cue",
			"",
			true,
		},
		{
			"err: file is not in computed dir",
			"services/foo/one/input.cue",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kuesta.ParseServiceComputedFilePath(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
				assert.Nil(t, err)
			}
		})
	}
}

func TestNewDevicePathList(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		dir := t.TempDir()
		testhelper.ExitOnErr(t, os.MkdirAll(filepath.Join(dir, "devices", "device1"), 0o750))
		testhelper.ExitOnErr(t, os.MkdirAll(filepath.Join(dir, "devices", "device2"), 0o750))
		testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "dummy"), []byte("dummy")))

		paths, err := kuesta.NewDevicePathList(dir)
		assert.Nil(t, err)
		assert.Len(t, paths, 2)
		for _, p := range paths {
			assert.Contains(t, []string{"device1", "device2"}, p.Device)
		}
	})

	t.Run("ok: no item", func(t *testing.T) {
		dir := t.TempDir()
		testhelper.ExitOnErr(t, os.MkdirAll(filepath.Join(dir, "devices"), 0o750))

		paths, err := kuesta.NewDevicePathList(dir)
		assert.Nil(t, err)
		assert.Len(t, paths, 0)
	})

	t.Run("err: no root", func(t *testing.T) {
		dir := t.TempDir()

		_, err := kuesta.NewDevicePathList(dir)
		assert.Error(t, err)
	})
}
