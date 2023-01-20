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

package logger_test

import (
	"context"
	"testing"

	"github.com/nttcom/kuesta/internal/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestConvertLevel(t *testing.T) {
	tests := []struct {
		given uint8
		want  zapcore.Level
	}{
		{0, zapcore.WarnLevel},
		{1, zapcore.InfoLevel},
		{2, zapcore.DebugLevel},
		{3, zapcore.DebugLevel},
	}

	for _, tt := range tests {
		assert.Equal(t, logger.ConvertLevel(tt.given), tt.want)
	}
}

func TestFromContext(t *testing.T) {
	want := logger.NewLogger()
	ctx := logger.WithLogger(context.Background(), want)
	assert.Equal(t, want, logger.FromContext(ctx))
}

func TestSetup(t *testing.T) {
	core := logger.NewLogger().Desugar().Core()
	assert.Equal(t, false, core.Enabled(zapcore.DebugLevel))

	logger.Setup(true, 2)
	core = logger.NewLogger().Desugar().Core()
	assert.Equal(t, true, core.Enabled(zapcore.DebugLevel))
	logger.SetDefault()
}
