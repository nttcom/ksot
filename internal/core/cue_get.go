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
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nttcom/kuesta/internal/logger"
	"github.com/nttcom/kuesta/internal/validator"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/modfile"
)

type CueGetCfg struct {
	RootCfg

	FilePath string
}

// Validate validates exposed fields according to the `validate` tag.
func (c *CueGetCfg) Validate() error {
	if filepath.Ext(c.FilePath) != ".go" {
		return fmt.Errorf("target is not go file")
	}
	return validator.Validate(c)
}

// Mask returns the copy whose sensitive data are masked.
func (c *CueGetCfg) Mask() *CueGetCfg {
	cc := *c
	cc.RootCfg = *c.RootCfg.Mask()
	return &cc
}

// RunCueGet runs the main process of the `cue get` command.
func RunCueGet(ctx context.Context, cfg *CueGetCfg) error {
	l := logger.FromContext(ctx)
	l.Debugw("cue get called", "config", cfg.Mask())
	return RunCueGetImpl(ctx, cfg.FilePath, execCueGet)
}

func RunCueGetImpl(ctx context.Context, path string, getter CueGetter) error {
	_ = WriterFromContext(ctx)

	if err := validateIsModuleRoot(); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	in, err := loadInput(path)
	if err != nil {
		return fmt.Errorf("load given go file: %w", err)
	}

	out, err := setupOutput(path)
	if err != nil {
		return fmt.Errorf("setup output file: %w", err)
	}
	defer out.Close()

	if err := ConvertMapKeyToString(path, in, out); err != nil {
		return fmt.Errorf("convert map key to string: %w", err)
	}

	outDir, _ := getOutFilePath(path)
	modPath, err := resolveModuleName()
	if err != nil {
		return fmt.Errorf("resolve module name: %w", err)
	}

	if err := getter.Exec(modPath, outDir); err != nil {
		return fmt.Errorf("execute cue get: %w", err)
	}
	return nil
}

// ConvertMapKeyToString converts all go structs in the given go file to the structure supported by kuesta.
// It extracts go structs and converts all map keys to string if the go struct contains maps with non-string key.
func ConvertMapKeyToString(path string, in io.Reader, out io.Writer) error {
	var v ast.Visitor
	v = VisitFunc(func(n ast.Node) ast.Visitor {
		if n == nil {
			return v
		}
		if _, ok := n.(*ast.MapType); !ok {
			// continue to next node using the same visitor
			return v
		}

		called := false
		w := VisitFunc(func(n ast.Node) ast.Visitor {
			if called {
				return nil
			}
			called = true
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name != "string" {
					ident.Name = "string"
				}
			}
			return nil
		})
		return w
	})

	_, filename := filepath.Split(path)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, in, 0)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "package %s", f.Name)
	for _, d := range f.Decls {
		if _, ok := d.(*ast.GenDecl); !ok {
			continue
		}
		ast.Walk(v, d)
		s, err := formatDecl(d)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "\n\n%s", s)
	}
	return nil
}

func validateIsModuleRoot() error {
	if _, err := os.Stat("go.mod"); err != nil {
		return errors.WithStack(fmt.Errorf("must run at the go module root where go.mod placed"))
	}
	return nil
}

func loadInput(path string) (io.Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return bytes.NewReader(data), nil
}

func setupOutput(path string) (*os.File, error) {
	outDir, outFile := getOutFilePath(path)
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return nil, errors.WithStack(err)
	}
	out, err := os.OpenFile(filepath.Join(outDir, outFile), os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return out, nil
}

type CueGetter interface {
	Exec(modPath, outDir string) error
}

type CueGetFunc func(modPath, outDir string) error

func (fn CueGetFunc) Exec(modPath, outDir string) error {
	return fn(modPath, outDir)
}

var execCueGet = CueGetFunc(func(modPath, outDir string) error {
	script := fmt.Sprintf(`
# Init cue.mod if not setup yet
if [[ ! -d cue.mod ]]; then
    cue mod init %s 
fi
# Generate cue type defs
cue get go --local %s
`, modPath, filepath.Join(modPath, outDir))

	if err := exec.Command("/bin/bash", "-c", script).Run(); err != nil {
		return errors.WithStack(err)
	}
	return nil
})

func getOutFilePath(path string) (string, string) {
	dir, filename := filepath.Split(path)
	trimmed := strings.TrimSuffix(dir, "/")
	dir = filepath.Join("types", trimmed)
	return dir, filename
}

func resolveModuleName() (string, error) {
	buf, err := os.ReadFile("go.mod")
	if err != nil {
		return "", errors.WithStack(err)
	}
	modPath := modfile.ModulePath(buf)
	if modPath == "" {
		return "", errors.WithStack(fmt.Errorf("module name is not found: the go.mod format might be invalid"))
	}
	return modPath, nil
}

type VisitFunc func(n ast.Node) ast.Visitor

func (v VisitFunc) Visit(n ast.Node) ast.Visitor {
	return v(n)
}

func formatDecl(f any) (io.Writer, error) {
	buf := &bytes.Buffer{}
	fset := token.NewFileSet()
	if err := format.Node(buf, fset, f); err != nil {
		return nil, err
	}
	return buf, nil
}
