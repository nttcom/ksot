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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/device-operator/internal/model"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/openconfig/ygot/ygot"
	"github.com/pkg/errors"
)

func main() {
	flag.Usage = func() {
		usage := `Convert cue data to RFC7951-style JSON.

Usage:
  converter [flags]

Flags:
`
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	path := validateArgs(args)
	convert(path)
}

func validateArgs(args []string) string {
	if len(args) == 0 {
		log.Fatal("target file does not specified")
	}
	path := args[0]
	if filepath.Ext(path) != ".cue" {
		log.Fatal("given file is not CUE file")
	}
	return path
}

func decodeCueBytes(cctx *cue.Context, bytes []byte) (*model.Device, error) {
	val, err := kcue.NewValueFromBytes(cctx, bytes)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var o model.Device
	if err := val.Decode(&o); err != nil {
		return nil, errors.WithStack(err)
	}
	return &o, nil
}

func convert(path string) {
	data, err := os.ReadFile(path)
	mustNil(err)
	cctx := cuecontext.New()
	d, err := decodeCueBytes(cctx, data)
	mustNil(err)
	jsonTree, err := ygot.ConstructIETFJSON(d, &ygot.RFC7951JSONConfig{AppendModuleName: true})
	mustNil(err)
	jsonDump, err := json.Marshal(jsonTree)
	mustNil(err)
	fmt.Printf("%s\n", jsonDump)
}

func mustNil(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
