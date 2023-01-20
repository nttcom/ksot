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

package gitrepo_test

import (
	"testing"

	"github.com/nttcom/kuesta/internal/gitrepo"
	"github.com/stretchr/testify/assert"
)

func TestNewGitHubClient(t *testing.T) {
	token := ""

	tests := []struct {
		name    string
		given   string
		wantRet bool
	}{
		{
			"ok",
			"https://github.com/hrk091/kuesta-testdata",
			true,
		},
		{
			"err: incorrect git repo",
			"https://not.exist.com/hrk091/kuesta-testdata",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gitrepo.NewGitHubClient(tt.given, token)
			if tt.wantRet {
				assert.NotNil(t, c)
			} else {
				assert.Nil(t, c)
			}
		})
	}
}

func TestNewRepoRef(t *testing.T) {
	tests := []struct {
		name    string
		given   string
		want    gitrepo.GitRepoRef
		wantErr bool
	}{
		{
			"ok",
			"https://github.com/hrk091/kuesta-testdata",
			gitrepo.GitRepoRef{
				Owner: "hrk091",
				Name:  "kuesta-testdata",
			},
			false,
		},
		{
			"ok: repoName including slash",
			"https://github.com/hrk091/kuesta/testdata",
			gitrepo.GitRepoRef{
				Owner: "hrk091",
				Name:  "kuesta/testdata",
			},
			false,
		},
		{
			"err: incorrect git repo",
			"https://not.exist.com/hrk091/kuesta-testdata",
			gitrepo.GitRepoRef{},
			true,
		},
		{
			"err: repoName not given",
			"https://not.exist.com/hrk091/",
			gitrepo.GitRepoRef{},
			true,
		},
		{
			"err: org not given",
			"https://not.exist.com/",
			gitrepo.GitRepoRef{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gitrepo.NewRepoRef(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
