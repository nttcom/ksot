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
	"context"
	"fmt"

	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/internal/logger"
	"github.com/nttcom/kuesta/internal/validator"
	"github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/kuesta"
)

type ServiceCompileCfg struct {
	RootCfg

	Service string   `validate:"required"`
	Keys    []string `validate:"gt=0,dive,required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *ServiceCompileCfg) Validate() error {
	return validator.Validate(c)
}

// Mask returns the copy whose sensitive data are masked.
func (c *ServiceCompileCfg) Mask() *ServiceCompileCfg {
	cc := *c
	cc.RootCfg = *c.RootCfg.Mask()
	return &cc
}

// RunServiceCompile runs the main process of the `service compile` command.
func RunServiceCompile(ctx context.Context, cfg *ServiceCompileCfg) error {
	l := logger.FromContext(ctx)
	l.Debugw("service compile called", "config", cfg.Mask())

	cctx := cuecontext.New()

	sp := kuesta.ServicePath{
		RootDir: cfg.ConfigRootPath,
		Service: cfg.Service,
		Keys:    cfg.Keys,
	}
	if err := sp.Validate(); err != nil {
		return fmt.Errorf("validate ServicePath: %w", err)
	}

	buf, err := sp.ReadServiceInput()
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}
	inputVal, err := cue.NewValueFromBytes(cctx, buf)
	if err != nil {
		return fmt.Errorf("load input file: %w", err)
	}

	transformer, err := sp.ReadServiceTransform(cctx)
	if err != nil {
		return fmt.Errorf("load transform file: %w", err)
	}
	it, err := transformer.Apply(inputVal)
	if err != nil {
		return fmt.Errorf("apply transform: %w", err)
	}

	for it.Next() {
		device := it.Label()
		buf, err := kuesta.NewDevice(it.Value()).Config()
		if err != nil {
			return fmt.Errorf("extract device config: %w", err)
		}

		if err := sp.WriteServiceComputedFile(device, buf); err != nil {
			return fmt.Errorf("save partial device config: %w", err)
		}
	}

	return nil
}
