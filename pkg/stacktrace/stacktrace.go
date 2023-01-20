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

package stacktrace

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Show shows the stacktrace of the original error only.
func Show(w io.Writer, err error) {
	if st := Get(err); st != "" {
		fmt.Fprintf(w, "StackTrace: %s\n\n", st)
	}
}

// Get returns the stacktrace of the original error only.
func Get(err error) string {
	st := bottomStackTrace(err)
	if st != nil {
		return fmt.Sprintf("%+v", st.StackTrace())
	}
	return ""
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func bottomStackTrace(err error) stackTracer {
	nestedErr := errors.Unwrap(err)
	if nestedErr != nil {
		if st := bottomStackTrace(nestedErr); st != nil {
			return st
		}
	}
	// type check after checking all nested errors are not stackTracer
	if e, ok := err.(stackTracer); ok { // nolint
		return e
	}
	return nil
}
