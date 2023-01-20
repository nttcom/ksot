//go:build test_github

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
	"context"
	"os"
	"testing"

	"github.com/nttcom/kuesta/internal/gitrepo"
	"github.com/stretchr/testify/assert"
)

func TestGitHubClientImpl_HealthCheck(t *testing.T) {
	repo := "https://github.com/hrk091/kuesta-testdata"
	c := gitrepo.NewGitHubClient(repo, os.Getenv("GITHUB_TOKEN"))
	err := c.HealthCheck()
	assert.Nil(t, err)
}

func TestGitHubClientImpl_CreatePullRequest(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")

	type given struct {
		repo    string
		payload gitrepo.GitPullRequestPayload
	}

	var tests = []struct {
		name string
		given
		wantErr bool
	}{
		{
			"ok",
			given{
				"https://github.com/hrk091/kuesta-testdata",
				gitrepo.GitPullRequestPayload{
					HeadRef: "github-client-test",
					BaseRef: "main",
					Title:   "github-client-test",
					Body:    "github-client-test",
				},
			},
			false,
		},
		{
			"err: pr-already-created",
			given{
				"https://github.com/hrk091/kuesta-testdata",
				gitrepo.GitPullRequestPayload{
					HeadRef: "pr-already-created",
					BaseRef: "main",
					Title:   "github-client-test",
					Body:    "github-client-test",
				},
			},
			true,
		},
		{
			"err: repository not found",
			given{
				"https://github.com/hrk091/NOT_EXIST",
				gitrepo.GitPullRequestPayload{
					HeadRef: "github-client-test",
					BaseRef: "main",
					Title:   "github-client-test",
					Body:    "github-client-test",
				},
			},
			true,
		},
		{
			"err: base branch not found",
			given{
				"https://github.com/hrk091/kuesta-testdata",
				gitrepo.GitPullRequestPayload{
					HeadRef: "github-client-test",
					BaseRef: "NOT_EXIST",
					Title:   "github-client-test",
					Body:    "github-client-test",
				},
			},
			true,
		},
		{
			"err: head branch not found",
			given{
				"https://github.com/hrk091/kuesta-testdata",
				gitrepo.GitPullRequestPayload{
					HeadRef: "NOT_EXIST",
					BaseRef: "github-client-test",
					Title:   "github-client-test",
					Body:    "github-client-test",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			c := gitrepo.NewGitHubClient(tt.repo, token)
			prNum, err := c.CreatePullRequest(ctx, tt.payload)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.True(t, prNum != 0)
			}
		})
	}
}
