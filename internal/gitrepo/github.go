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
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// NewRepoRef returns GitRepoRef from the remote hostname.
func NewRepoRef(repoURL string) (GitRepoRef, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return GitRepoRef{}, errors.WithStack(fmt.Errorf("parse git repository URL: %w", err))
	}
	host := u.Hostname()
	if host != "github.com" {
		return GitRepoRef{}, errors.WithStack(fmt.Errorf("git repository host mismatched"))
	}
	items := strings.SplitN(u.Path, "/", 3)
	owner := items[1]
	repoName := items[2]
	if owner == "" || repoName == "" {
		return GitRepoRef{}, errors.WithStack(fmt.Errorf("invalid git repository URL: %s", repoURL))
	}

	return GitRepoRef{
		Owner: items[1],
		Name:  items[2],
	}, nil
}

func init() {
	gitClientConstructors = append(gitClientConstructors, NewGitHubClient)
}

var _ GitRepoClient = &GitHubClientImpl{}

// GitHubClientImpl implements GitRepoClient which works with GitHub.
type GitHubClientImpl struct {
	repo        GitRepoRef
	tokenSource oauth2.TokenSource
}

// NewGitHubClient creates new GitRepoClient which works with GitHub.
func NewGitHubClient(repoURL string, token string) GitRepoClient {
	repo, err := NewRepoRef(repoURL)
	if err != nil {
		return nil
	}
	if token == "" {
		return &GitHubClientImpl{repo: repo}
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return &GitHubClientImpl{repo, src}
}

func (g GitHubClientImpl) Kind() string {
	return "github"
}

func (g GitHubClientImpl) HealthCheck() error {
	ctx := context.Background()
	client := githubv4.NewClient(oauth2.NewClient(ctx, g.tokenSource))

	q := &healthQuery{}
	if err := client.Query(ctx, q, nil); err != nil {
		return errors.WithStack(fmt.Errorf("health check: %w", err))
	}
	return nil
}

func (g GitHubClientImpl) CreatePullRequest(ctx context.Context, payload GitPullRequestPayload) (prNum int, err error) {
	repoID, err := g.getRepositoryId(ctx, g.repo)
	if err != nil {
		return 0, err
	}

	client := githubv4.NewClient(oauth2.NewClient(ctx, g.tokenSource))
	m := &createPullRequestMutation{}
	if err := client.Mutate(ctx, m, createPullRequestInput(repoID, payload), nil); err != nil {
		return 0, errors.WithStack(fmt.Errorf("create PR: %w", err))
	}
	return m.CreatePullRequest.PullRequest.Number, nil
}

func (g *GitHubClientImpl) getRepositoryId(ctx context.Context, repo GitRepoRef) (githubv4.ID, error) {
	client := githubv4.NewClient(oauth2.NewClient(ctx, g.tokenSource))

	q := &getRepoQuery{}
	if err := client.Query(ctx, q, getRepoVariable(repo)); err != nil {
		return nil, errors.WithStack(fmt.Errorf("health check: %w", err))
	}
	return q.Repository.ID, nil
}
