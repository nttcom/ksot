//go:generate mockgen -package=mock -source=gitrepo.go -destination=mock/gitrepo.go

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

package gitrepo

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

type GitRepoRef struct {
	Owner string
	Name  string
}

type GitPullRequestPayload struct {
	HeadRef string
	BaseRef string
	Title   string
	Body    string
}

type GitRepoClient interface {
	// Kind returns the kind of git-repo client.
	Kind() string

	// HealthCheck sends a simple query with auth-token to confirm that the request reaches and is accepted by remote git-repository.
	HealthCheck() error

	// CreatePullRequest creates PullRequest with given parameters.
	CreatePullRequest(ctx context.Context, payload GitPullRequestPayload) (prNum int, err error)
}

type NewGitClientFunc func(repoURL string, token string) GitRepoClient

var gitClientConstructors []NewGitClientFunc

// NewGitRepoClient creates new GitRepoClient for the given git repository kind.
func NewGitRepoClient(repoURL string, token string) (GitRepoClient, error) {
	for _, create := range gitClientConstructors {
		if c := create(repoURL, token); c != nil {
			return c, nil
		}
	}
	return nil, errors.WithStack(fmt.Errorf("resolve correspoinding GitRepoClient: unknown git-repo host: path=`%s`", repoURL))
}

// ReplaceGitClientConstructors replaces gitClientConstructors with the given one, TEST USE ONLY.
func ReplaceGitClientConstructors(c GitRepoClient) {
	fn := func(repoURL string, token string) GitRepoClient {
		return c
	}
	gitClientConstructors = []NewGitClientFunc{fn}
}
