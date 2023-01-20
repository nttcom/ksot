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
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/internal/validator"
)

type RootCfg struct {
	Verbose        uint8 `validate:"min=0,max=3"`
	Devel          bool
	ConfigRootPath string
	StatusRootPath string
	ConfigRepoUrl  string
	StatusRepoUrl  string
	GitTrunk       string
	GitRemote      string
	GitToken       string
	GitUser        string
	GitEmail       string
	PushToMain     bool
}

// Validate validates exposed fields according to the `validate` tag.
func (c *RootCfg) Validate() error {
	return validator.Validate(c)
}

// Mask returns the copy whose sensitive data are masked.
func (c *RootCfg) Mask() *RootCfg {
	cc := *c
	cc.GitToken = "***"
	return &cc
}

func (c *RootCfg) ConfigGitOptions() *gogit.GitOptions {
	return &gogit.GitOptions{
		RepoUrl:     c.ConfigRepoUrl,
		Path:        c.ConfigRootPath,
		TrunkBranch: c.GitTrunk,
		RemoteName:  c.GitRemote,
		Token:       c.GitToken,
		User:        c.GitUser,
		Email:       c.GitEmail,
	}
}

func (c *RootCfg) StatusGitOptions() *gogit.GitOptions {
	return &gogit.GitOptions{
		RepoUrl:     c.StatusRepoUrl,
		Path:        c.StatusRootPath,
		TrunkBranch: c.GitTrunk,
		RemoteName:  c.GitRemote,
		Token:       c.GitToken,
		User:        c.GitUser,
		Email:       c.GitEmail,
	}
}
