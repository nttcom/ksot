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

package kuesta

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	cueTypeStrInput    = "#Input"
	cueTypeStrTemplate = "#Template"
	cuePathInput       = "input"
	cuePathOutput      = "output"
	cuePathDevice      = "devices"
	cuePathConfig      = "config"
)

type ServiceMeta struct {
	Kind         string `yaml:"kind,omitempty"`         // Kind of the model.
	Organization string `yaml:"organization,omitempty"` // Organization publishing the model.
	Version      string `yaml:"version,omitempty"`      // Semantic version of the model.
	Description  string `yaml:"description"`
}

// ModelData returns the gnmi.ModelData.
func (m *ServiceMeta) ModelData() *pb.ModelData {
	return &pb.ModelData{
		Name:         m.Kind,
		Organization: m.Organization,
		Version:      m.Version,
	}
}

// ReadServiceMeta returns ServiceMeta loaded from the metadata file on the given path.
func ReadServiceMeta(path string) (*ServiceMeta, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	var meta ServiceMeta
	if err := yaml.Unmarshal(buf, &meta); err != nil {
		return nil, errors.WithStack(err)
	}
	return &meta, nil
}

type ServiceTransformer struct {
	value cue.Value
}

// NewServiceTransformer creates ServiceTransformer with the given cue.Value.
func NewServiceTransformer(v cue.Value) *ServiceTransformer {
	return &ServiceTransformer{value: v}
}

// ReadServiceTransformer builds cue.Instance from the specified files and returns ServiceTransformer.
func ReadServiceTransformer(cctx *cue.Context, filepaths []string, dir string) (*ServiceTransformer, error) {
	v, err := kcue.NewValueWithInstance(cctx, filepaths, &load.Config{Dir: dir})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &ServiceTransformer{value: v}, nil
}

// Value returns the cue value contained.
func (t *ServiceTransformer) Value() cue.Value {
	return t.value
}

// Apply performs cue evaluation of transform.cue using given input.
// It returns cue.Iterator which iterates items including device name label and device config cue.Value.
func (t *ServiceTransformer) Apply(input cue.Value) (*cue.Iterator, error) {
	cctx := t.value.Context()
	template := cctx.CompileString(cueTypeStrTemplate, cue.Scope(t.value))
	if template.Err() != nil {
		return nil, errors.WithStack(template.Err())
	}
	filled := template.FillPath(cue.ParsePath(cuePathInput), input)
	if filled.Err() != nil {
		return nil, errors.WithStack(filled.Err())
	}

	filledIn := filled.LookupPath(cue.ParsePath(cuePathInput))
	if err := filledIn.Validate(cue.Concrete(true)); err != nil {
		return nil, errors.WithStack(err)
	}

	filledOut := filled.LookupPath(cue.ParsePath(cuePathOutput))
	if err := filledOut.Validate(); err != nil {
		return nil, errors.WithStack(err)
	}
	out := filledOut.Eval()
	if out.Err() != nil {
		return nil, errors.WithStack(out.Err())
	}
	it, err := out.LookupPath(cue.ParsePath(cuePathDevice)).Fields()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return it, nil
}

// ConvertInputType converts the type of given input according to the type defined as #Input in transform.cue.
func (t *ServiceTransformer) ConvertInputType(input map[string]string) (map[string]any, error) {
	converted := map[string]any{}
	for k, v := range input {
		kind := kcue.CueKindOf(t.value, fmt.Sprintf("%s.%s", cueTypeStrInput, k))
		if kind == cue.BottomKind {
			return nil, fmt.Errorf("key=%s is not defined in input types", k)
		}
		convert, err := kcue.NewStrConvFunc(kind)
		if err != nil {
			return nil, fmt.Errorf("the type of key=%s must be in string|int|float|bool|null: %w", k, err)
		}
		vv, err := convert(v)
		if err != nil {
			return nil, fmt.Errorf("type mismatch: key=%s, value=%s: %w", k, v, err)
		}
		converted[k] = vv
	}
	return converted, nil
}

// InputKeys returns unique keys of input specified at the kuesta tag.
func (t *ServiceTransformer) InputKeys() ([]string, error) {
	it, err := t.value.LookupPath(cue.ParsePath(cueTypeStrInput)).Fields()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	keys := map[int]string{}

	for it.Next() {
		name := it.Label()
		tag, err := kcue.CueKuestaTagOf(t.value, fmt.Sprintf("%s.%s", cueTypeStrInput, name))
		if err != nil {
			return nil, fmt.Errorf("get kuesta tag of field `%s`: %w", name, err)
		}
		kv := strings.SplitN(tag, "=", 2)
		if kv[0] != "key" {
			continue
		}
		seq, err := strconv.Atoi(kv[1])
		if err != nil {
			return nil, fmt.Errorf("convert unique key seq on kuesta tag `%s`: %w", name, err)
		}
		if _, alreadySet := keys[seq]; alreadySet {
			return nil, errors.WithStack(fmt.Errorf("unique key number duplicated: key=%d", seq))
		}
		keys[seq] = name
	}
	if len(keys) == 0 {
		return nil, errors.WithStack(fmt.Errorf("at least one key is needed"))
	}

	var ret []string
	for i := 0; i < len(keys); i++ {
		v, ok := keys[i+1]
		if !ok {
			return nil, errors.WithStack(fmt.Errorf("unique keys must be the sequence starting from 1"))
		}
		ret = append(ret, v)
	}
	return ret, nil
}

type Device struct {
	value cue.Value
}

// NewDevice creates Device with the given cue.Value.
func NewDevice(v cue.Value) *Device {
	return &Device{value: v}
}

// NewDeviceFromBytes creates Device from the given encoded cue bytes.
func NewDeviceFromBytes(cctx *cue.Context, buf []byte) (*Device, error) {
	v, err := kcue.NewValueFromBytes(cctx, buf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Device{value: v}, nil
}

// ReadDevice reads the cue file at the specified path and creates Device.
func ReadDevice(cctx *cue.Context, path string) (*Device, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return NewDeviceFromBytes(cctx, buf)
}

func (d *Device) Value() cue.Value {
	return d.value
}

// Config returns the device config bytes.
func (d *Device) Config() ([]byte, error) {
	cfg := d.value.LookupPath(cue.ParsePath(cuePathConfig))
	if cfg.Err() != nil {
		return nil, errors.WithStack(cfg.Err())
	}
	return kcue.FormatCue(cfg, cue.Final())
}
