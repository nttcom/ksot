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

package kuesta

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/nttcom/kuesta/internal/file"
	"github.com/nttcom/kuesta/internal/validator"
	"github.com/pkg/errors"
)

const (
	DirServices         = "services"
	DirDevices          = "devices"
	DirComputed         = "computed"
	FileInputCue        = "input.cue"
	FileTransformCue    = "transform.cue"
	FileServiceMetaYaml = "metadata.yaml"
	FileConfigCue       = "config.cue"
	FileActualConfigCue = "actual_config.cue"
)

type PathOpt string

const (
	ExcludeRoot PathOpt = ""
	IncludeRoot PathOpt = "INCLUDE_ROOT"
)

var _sep = string(filepath.Separator)

type ServicePath struct {
	RootDir string `validate:"required"`

	Service string
	Keys    []string `validate:"dive,required"`
}

// NewServicePathList returns the slice of ServicePath placed in the given root dir.
func NewServicePathList(dir string) ([]*ServicePath, error) {
	sp := ServicePath{RootDir: dir}
	path := sp.ServiceDirPath(IncludeRoot)
	services, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("read services dir: %w", err))
	}
	var spList []*ServicePath
	for _, d := range services {
		if !d.IsDir() {
			continue
		}
		spList = append(spList, &ServicePath{RootDir: dir, Service: d.Name()})
	}
	return spList, nil
}

// Validate validates exposed fields according to the `validate` tag.
func (p *ServicePath) Validate() error {
	return validator.Validate(p)
}

// RootPath returns the path to repository root.
func (p *ServicePath) RootPath() string {
	return filepath.FromSlash(p.RootDir)
}

func (p *ServicePath) serviceDirElem() []string {
	return []string{DirServices}
}

func (p *ServicePath) serviceColElem() []string {
	return append(p.serviceDirElem(), p.Service)
}

func (p *ServicePath) servicePathElem() []string {
	return append(p.serviceColElem(), p.Keys...)
}

func (p *ServicePath) serviceComputedPathElem() []string {
	return append(p.servicePathElem(), DirComputed)
}

func (p *ServicePath) addRoot(path string, t PathOpt) string {
	if t == ExcludeRoot {
		return path
	} else {
		return filepath.Join(p.RootPath(), path)
	}
}

// ServiceDirPath returns the path to the service directory.
func (p *ServicePath) ServiceDirPath(t PathOpt) string {
	return p.addRoot(filepath.Join(p.serviceDirElem()...), t)
}

// ServicePath returns the path to the specified service.
func (p *ServicePath) ServicePath(t PathOpt) string {
	return p.addRoot(filepath.Join(p.servicePathElem()...), t)
}

// ServiceInputPath returns the path to the specified service's input file.
func (p *ServicePath) ServiceInputPath(t PathOpt) string {
	el := append(p.servicePathElem(), FileInputCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceInput loads the specified service's input file.
func (p *ServicePath) ReadServiceInput() ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceInputPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// WriteServiceInputFile writes the supplied service's input file.
func (p *ServicePath) WriteServiceInputFile(buf []byte) error {
	return file.WriteFileWithMkdir(p.ServiceInputPath(IncludeRoot), buf)
}

// ServiceTransformPath returns the path to the specified service's transform file.
func (p *ServicePath) ServiceTransformPath(t PathOpt) string {
	el := append(p.serviceColElem(), FileTransformCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceTransform loads the specified service's transform file.
func (p *ServicePath) ReadServiceTransform(cctx *cue.Context) (*ServiceTransformer, error) {
	return ReadServiceTransformer(cctx, []string{p.ServiceTransformPath(ExcludeRoot)}, p.RootPath())
}

// ServiceComputedDirPath returns the path to the specified service's computed dir.
func (p *ServicePath) ServiceComputedDirPath(t PathOpt) string {
	return p.addRoot(filepath.Join(p.serviceComputedPathElem()...), t)
}

// ServiceComputedPath returns the path to the specified service's computed result of given device.
func (p *ServicePath) ServiceComputedPath(device string, t PathOpt) string {
	el := append(p.serviceComputedPathElem(), fmt.Sprintf("%s.cue", device))
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceComputedFile loads the partial device config computed from specified service.
func (p *ServicePath) ReadServiceComputedFile(device string) ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceComputedPath(device, IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// WriteServiceComputedFile writes the partial device config computed from service to the corresponding computed dir.
func (p *ServicePath) WriteServiceComputedFile(device string, buf []byte) error {
	return file.WriteFileWithMkdir(p.ServiceComputedPath(device, IncludeRoot), buf)
}

// ServiceMetaPath returns the path to the service meta.
func (p *ServicePath) ServiceMetaPath(t PathOpt) string {
	el := append(p.serviceColElem(), FileServiceMetaYaml)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceMeta loads the service meta.
func (p *ServicePath) ReadServiceMeta() (*ServiceMeta, error) {
	meta, err := ReadServiceMeta(p.ServiceMetaPath(IncludeRoot))
	if err != nil {
		return nil, err
	}
	if meta == nil {
		meta = &ServiceMeta{}
	}
	if meta.Kind == "" {
		meta.Kind = p.Service
	} else if meta.Kind != p.Service {
		return nil, fmt.Errorf("kind and service path are mismatched: kind=%s, path=%s", meta.Kind, p.Service)
	}
	return meta, nil
}

type DevicePath struct {
	RootDir string `validate:"required"`

	Device string
}

// NewDevicePathList returns the slice of DevicePath placed in the given root dir.
func NewDevicePathList(dir string) ([]*DevicePath, error) {
	dp := DevicePath{RootDir: dir}
	path := dp.DeviceDirPath(IncludeRoot)
	devices, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read devices dir: %w", err)
	}
	var dpList []*DevicePath
	for _, d := range devices {
		if !d.IsDir() {
			continue
		}
		dpList = append(dpList, &DevicePath{RootDir: dir, Device: d.Name()})
	}
	return dpList, nil
}

// Validate validates exposed fields according to the `validate` tag.
func (p *DevicePath) Validate() error {
	return validator.Validate(p)
}

// RootPath returns the path to repository root.
func (p *DevicePath) RootPath() string {
	return filepath.FromSlash(p.RootDir)
}

func (p *DevicePath) deviceDirElem() []string {
	return []string{DirDevices}
}

func (p *DevicePath) devicePathElem() []string {
	return append(p.deviceDirElem(), p.Device)
}

func (p *DevicePath) addRoot(path string, t PathOpt) string {
	if t == ExcludeRoot {
		return path
	} else {
		return filepath.Join(p.RootPath(), path)
	}
}

// DeviceDirPath returns the path to the devices directory.
func (p *DevicePath) DeviceDirPath(t PathOpt) string {
	return p.addRoot(filepath.Join(p.deviceDirElem()...), t)
}

// DevicePath returns the path to the devices directory.
func (p *DevicePath) DevicePath(t PathOpt) string {
	return p.addRoot(filepath.Join(p.devicePathElem()...), t)
}

// DeviceConfigPath returns the path to specified device config.
func (p *DevicePath) DeviceConfigPath(t PathOpt) string {
	el := append(p.devicePathElem(), FileConfigCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadDeviceConfigFile loads the device config.
func (p *DevicePath) ReadDeviceConfigFile() ([]byte, error) {
	buf, err := os.ReadFile(p.DeviceConfigPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// WriteDeviceConfigFile writes the merged device config to the corresponding device dir.
func (p *DevicePath) WriteDeviceConfigFile(buf []byte) error {
	return file.WriteFileWithMkdir(p.DeviceConfigPath(IncludeRoot), buf)
}

// CheckSum returns the SHA256 checksum of the device config.
func (p *DevicePath) CheckSum() (string, error) {
	f, err := os.Open(p.DeviceConfigPath(IncludeRoot))
	if err != nil {
		return "", errors.WithStack(err)
	}
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", errors.WithStack(err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))
	return checksum, nil
}

// DeviceActualConfigPath returns the path to specified device actual config.
func (p *DevicePath) DeviceActualConfigPath(t PathOpt) string {
	el := append(p.devicePathElem(), FileActualConfigCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadActualDeviceConfigFile loads the device actual config.
func (p *DevicePath) ReadActualDeviceConfigFile() ([]byte, error) {
	buf, err := os.ReadFile(p.DeviceActualConfigPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// ReadServiceMetaAll loads all service meta stored in the git repo.
func ReadServiceMetaAll(dir string) ([]*ServiceMeta, error) {
	var mlist []*ServiceMeta
	spList, err := NewServicePathList(dir)
	if err != nil {
		return nil, err
	}
	for _, sp := range spList {
		if meta, err := sp.ReadServiceMeta(); err == nil {
			mlist = append(mlist, meta)
		}
	}
	return mlist, nil
}

// ParseServiceInputPath parses service model `input.cue` filepath and returns its service and keys.
func ParseServiceInputPath(path string) (string, []string, error) {
	if !isServiceInputPath(path) {
		return "", nil, errors.WithStack(fmt.Errorf("invalid service input path: %s", path))
	}
	dir, _ := filepath.Split(path)
	dirElem := strings.Split(strings.TrimRight(dir, _sep), _sep)
	return dirElem[1], dirElem[2:], nil
}

func isServiceInputPath(path string) bool {
	dir, file := filepath.Split(path)
	dirElem := strings.Split(dir, string(filepath.Separator))
	if dirElem[0] != "services" {
		return false
	}
	if file != "input.cue" {
		return false
	}
	return true
}

// ParseServiceComputedFilePath parses service computed filepath and returns its device name.
func ParseServiceComputedFilePath(path string) (string, error) {
	if !isServiceComputedFilePath(path) {
		return "", errors.WithStack(fmt.Errorf("invalid service computed path: %s", path))
	}
	return getFileNameNoExt(path), nil
}

func isServiceComputedFilePath(path string) bool {
	dir, _ := filepath.Split(path)
	dirElem := strings.Split(strings.TrimRight(dir, _sep), _sep)
	if dirElem[0] != DirServices {
		return false
	}
	if dirElem[len(dirElem)-1] != DirComputed {
		return false
	}
	return true
}

func getFileNameNoExt(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}
