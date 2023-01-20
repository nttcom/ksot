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

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/internal/logger"
	"github.com/nttcom/kuesta/internal/validator"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/kuesta"
)

type DeviceCompositeCfg struct {
	RootCfg

	Device string `validate:"required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *DeviceCompositeCfg) Validate() error {
	return validator.Validate(c)
}

// Mask returns the copy whose sensitive data are masked.
func (c *DeviceCompositeCfg) Mask() *DeviceCompositeCfg {
	cc := *c
	cc.RootCfg = *c.RootCfg.Mask()
	return &cc
}

// RunDeviceComposite runs the main process of the `device composite` command.
func RunDeviceComposite(ctx context.Context, cfg *DeviceCompositeCfg) error {
	l := logger.FromContext(ctx)
	l.Debugw("device composite called", "config", cfg.Mask())

	cctx := cuecontext.New()
	sp := kuesta.ServicePath{RootDir: cfg.ConfigRootPath}
	dp := kuesta.DevicePath{RootDir: cfg.ConfigRootPath, Device: cfg.Device}

	files, err := CollectPartialDeviceConfig(sp.ServiceDirPath(kuesta.IncludeRoot), cfg.Device)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}
	l.Debugw("merging partial device configs", "files", files)

	// composite all partial device configs into one CUE instance
	deviceConfig, err := kcue.NewValueWithInstance(cctx, files, nil)
	if err != nil {
		return fmt.Errorf("composite files: %w", err)
	}

	buf, err := kcue.FormatCue(deviceConfig, cue.Concrete(true))
	if err != nil {
		return fmt.Errorf("format merged config: %w", err)
	}

	if err := dp.WriteDeviceConfigFile(buf); err != nil {
		return fmt.Errorf("write merged config: %w", err)
	}

	return nil
}
