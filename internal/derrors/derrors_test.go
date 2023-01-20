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

package derrors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/nttcom/kuesta/internal/derrors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	grpcErrMsg = "error for user"
	origErr    = errors.New("original error")
	grpcErr    = status.New(codes.Internal, grpcErrMsg).Err()
)

func TestGRPCErrorf(t *testing.T) {
	got := derrors.GRPCErrorf(origErr, codes.Internal, grpcErrMsg)
	want := status.New(codes.Internal, grpcErrMsg)

	var we *derrors.GRPCWrapError
	ok := errors.As(got, &we)
	assert.True(t, ok)
	assert.Equal(t, want, we.Status())
	assert.Equal(t, origErr.Error(), we.Error())
}

func TestToGRPCError(t *testing.T) {
	wrapErr := derrors.GRPCErrorf(origErr, codes.Internal, grpcErrMsg)
	moreWrapErr := fmt.Errorf("wrapped: %w", wrapErr)
	wrapGrpcErr := fmt.Errorf("wrapped: %w", grpcErr)

	tests := []struct {
		name        string
		given       error
		wantGRPCErr error
		wantWrapErr error
	}{
		{
			"ok: GRPCWrapError",
			wrapErr,
			grpcErr,
			wrapErr,
		},
		{
			"ok: wrapped GRPCWrapError",
			moreWrapErr,
			grpcErr,
			moreWrapErr,
		},
		{
			"ok: GRPCError",
			grpcErr,
			grpcErr,
			nil,
		},
		{
			"ok: wrapped GRPCError",
			wrapGrpcErr,
			grpcErr,
			nil,
		},
		{
			"ok: non GRPCError",
			origErr,
			origErr,
			origErr,
		},
		{
			"ok: nil",
			nil,
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e1, e2 := derrors.ToGRPCError(tt.given)
			assert.Equal(t, tt.wantGRPCErr, e1)
			assert.Equal(t, tt.wantWrapErr, e2)
		})
	}
}
