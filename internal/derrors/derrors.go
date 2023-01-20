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

package derrors

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ error = &GRPCWrapError{}

type GRPCWrapError struct {
	s    *status.Status
	werr error
}

// GRPCErrorf creates error containing pair errors: grpc.Status with given code and message, and underlying error.
func GRPCErrorf(err error, c codes.Code, format string, a ...interface{}) error {
	return &GRPCWrapError{
		s:    status.Newf(c, format, a...),
		werr: err,
	}
}

func (e *GRPCWrapError) Error() string {
	return e.werr.Error()
}

func (e *GRPCWrapError) Unwrap() error {
	return e.werr
}

func (e *GRPCWrapError) Status() *status.Status {
	return e.s
}

// ToGRPCError separates grpc.Error and underlying error.
func ToGRPCError(err error) (error, error) {
	if err == nil {
		return nil, nil
	}
	if we := (*GRPCWrapError)(nil); errors.As(err, &we) {
		return we.s.Err(), err
	}
	var ge interface {
		GRPCStatus() *status.Status
	}
	if errors.As(err, &ge) {
		return ge.GRPCStatus().Err(), nil
	}
	return err, err
}
