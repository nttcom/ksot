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

package logger

import (
	"context"
	"fmt"

	"github.com/nttcom/kuesta/pkg/stacktrace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type _keyLogger struct{}

var (
	config     zap.Config
	rootLogger *zap.Logger
)

func init() {
	SetDefault()
}

func SetDefault() {
	config = zap.NewProductionConfig()
	rootLogger, _ = config.Build()
}

func Setup(isDevel bool, lvl uint8, opts ...zap.Option) {
	if isDevel {
		config = zap.NewDevelopmentConfig()
	}
	config.Level = zap.NewAtomicLevelAt(ConvertLevel(lvl))
	rootLogger, _ = config.Build(opts...)
}

func ConvertLevel(lvl uint8) zapcore.Level {
	if lvl < 3 {
		return zapcore.Level(1 - lvl)
	} else {
		return zapcore.DebugLevel
	}
}

func NewLogger() *zap.SugaredLogger {
	return rootLogger.Sugar()
}

func WithLogger(parent context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(parent, _keyLogger{}, logger)
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	v, ok := ctx.Value(_keyLogger{}).(*zap.SugaredLogger)
	if !ok {
		return NewLogger()
	} else {
		return v
	}
}

// ErrorWithStack shows error log along with its stacktrace.
func ErrorWithStack(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := FromContext(ctx).WithOptions(zap.AddCallerSkip(1))
	if st := stacktrace.Get(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}
