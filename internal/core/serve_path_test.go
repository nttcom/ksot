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
	"path/filepath"
	"testing"

	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

func TestGnmiPathConverter_Convert(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		prefix  *pb.Path
		path    *pb.Path
		setup   func(dir string)
		want    any
		wantErr bool
	}{
		{
			"ok: service",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				path := filepath.Join(dir, "services", "foo", "transform.cue")
				testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(path, []byte(`
#Input: {
	// kuesta:"key=1"
	bar:   string
	// kuesta:"key=2"
	baz:   int64
}`)))
			},
			&kuesta.ServicePath{
				RootDir: dir,
				Service: "foo",
				Keys:    []string{"one", "two"},
			},
			false,
		},
		{
			"ok: service with prefix",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				path := filepath.Join(dir, "services", "foo", "transform.cue")
				testhelper.ExitOnErr(t, testhelper.WriteFileWithMkdir(path, []byte(`
#Input: {
	// kuesta:"key=1"
	bar:   string
	// kuesta:"key=2"
	baz:   int64
}`)))
			},
			&kuesta.ServicePath{
				RootDir: dir,
				Service: "foo",
				Keys:    []string{"one", "two"},
			},
			false,
		},
		{
			"ok: device",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			&kuesta.DevicePath{
				RootDir: dir,
				Device:  "device1",
			},
			false,
		},
		{
			"ok: device with prefix",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			&kuesta.DevicePath{
				RootDir: dir,
				Device:  "device1",
			},
			false,
		},
		{
			"err: service meta not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: elem length is less than 2",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: invalid name",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: invalid service name",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: invalid device name",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(dir)
			}
			c := core.NewGnmiPathConverter(&core.ServeCfg{
				RootCfg: core.RootCfg{
					ConfigRootPath: dir,
					StatusRootPath: dir,
				},
			})
			got, err := c.Convert(tt.prefix, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				switch r := got.(type) {
				case core.ServicePathReq:
					assert.Equal(t, tt.want, r.Path())
				case core.DevicePathReq:
					assert.Equal(t, tt.want, r.Path())
				default:
					t.Fatalf("unexpected type: %T", got)
				}
			}
		})
	}
}
