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

package cmd_test

import (
	"os"
	"testing"

	"github.com/nttcom/kuesta/internal/cmd"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNewRootCmd(t *testing.T) {
	dummyToken := "dummy-git-token"
	testhelper.ExitOnErr(t, os.Setenv("KUESTA_GIT_TOKEN", dummyToken))

	dummyRootpath := "dummy-rootpath"
	testhelper.ExitOnErr(t, os.Setenv("KUESTA_CONFIG_ROOT_PATH", dummyRootpath))

	_ = cmd.NewRootCmd()
	assert.Equal(t, dummyToken, viper.GetString(cmd.FlagGitToken))
	assert.Equal(t, dummyRootpath, viper.GetString(cmd.FlagConfigRootPath))
}
