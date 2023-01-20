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
	"fmt"

	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
)

type PathReq interface {
	Type() PathType
	String() string
}

type ServicePathReq struct {
	path    *kuesta.ServicePath
	service string
	keys    map[string]string
}

func (ServicePathReq) Type() PathType {
	return PathTypeService
}

func (s ServicePathReq) String() string {
	return s.path.ServicePath(kuesta.ExcludeRoot)
}

func (s *ServicePathReq) Path() *kuesta.ServicePath {
	return s.path
}

func (s *ServicePathReq) Keys() map[string]string {
	return s.keys
}

type DevicePathReq struct {
	path   *kuesta.DevicePath
	device string
}

func (DevicePathReq) Type() PathType {
	return PathTypeDevice
}

func (s DevicePathReq) String() string {
	return s.path.DevicePath(kuesta.ExcludeRoot)
}

func (s DevicePathReq) Path() *kuesta.DevicePath {
	return s.path
}

type GnmiPathConverter struct {
	cfg *ServeCfg

	// meta caches service metadata
	// TODO clear cache periodically
	meta map[string]*kuesta.ServiceMeta
}

func NewGnmiPathConverter(cfg *ServeCfg) *GnmiPathConverter {
	return &GnmiPathConverter{
		cfg:  cfg,
		meta: map[string]*kuesta.ServiceMeta{},
	}
}

// Convert converts gNMI Path to PathReq.
func (c *GnmiPathConverter) Convert(prefix, path *gnmi.Path) (PathReq, error) {
	path = gnmiFullPath(prefix, path)
	elem := path.GetElem()
	if len(elem) < 2 {
		return nil, errors.WithStack(fmt.Errorf("path must have at least 2 elem"))
	}
	kindEl := elem[0]
	switch kindEl.GetName() {
	case kuesta.DirServices:
		return c.convertService(elem[1:])
	case kuesta.DirDevices:
		return c.convertDevice(elem[1:])
	default:
		return nil, errors.WithStack(fmt.Errorf("name of the first elem must be `%s` or `%s`", kuesta.DirServices, kuesta.DirDevices))
	}
}

func (c *GnmiPathConverter) convertService(elem []*gnmi.PathElem) (ServicePathReq, error) {
	svcEl := elem[0]
	if svcEl.GetName() != NodeService {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("name of second elem must be `%s`", NodeService))
	}
	elemKey := svcEl.GetKey()
	svcKind, ok := elemKey[KeyServiceKind]
	if !ok {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyServiceKind))
	}
	p := kuesta.ServicePath{RootDir: c.cfg.ConfigRootPath, Service: svcKind}

	cctx := cuecontext.New()
	tf, err := p.ReadServiceTransform(cctx)
	if err != nil {
		return ServicePathReq{}, fmt.Errorf("load service transform.cue: %w", err)
	}
	uniqKeys, err := tf.InputKeys()
	if err != nil {
		return ServicePathReq{}, fmt.Errorf("resolve service keys from transform.cue: %w", err)
	}

	keys := map[string]string{}
	for _, k := range uniqKeys {
		if v, ok := elemKey[k]; ok {
			keys[k] = v
			p.Keys = append(p.Keys, v)
		} else {
			return ServicePathReq{}, errors.WithStack(fmt.Errorf("key `%s` of service %s is not supplied in Request Path", k, svcKind))
		}
	}
	return ServicePathReq{path: &p, service: svcKind, keys: keys}, nil
}

func (c *GnmiPathConverter) convertDevice(elem []*gnmi.PathElem) (DevicePathReq, error) {
	svcEl := elem[0]
	if svcEl.GetName() != NodeDevice {
		return DevicePathReq{}, errors.WithStack(fmt.Errorf("name of second elem must be `%s`", NodeDevice))
	}
	keys := svcEl.GetKey()
	deviceName, ok := keys[KeyDeviceName]
	if !ok {
		return DevicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyDeviceName))
	}

	p := kuesta.DevicePath{RootDir: c.cfg.StatusRootPath, Device: deviceName}
	return DevicePathReq{path: &p, device: deviceName}, nil
}

func gnmiFullPath(prefix, path *gnmi.Path) *gnmi.Path {
	fullPath := &gnmi.Path{}
	if path.GetElem() != nil {
		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
	}
	return fullPath
}
