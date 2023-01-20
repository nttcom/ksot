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

package gogit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gogithttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/nttcom/kuesta/internal/util"
	"github.com/nttcom/kuesta/internal/validator"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

const (
	DefaultAuthUser    = "anonymous"
	DefaultTrunkBranch = "main"
	DefaultRemoteName  = "origin"
	DefaultGitUser     = "kuesta"
	DefaultGitEmail    = "kuesta@example.com"
)

type GitOptions struct {
	shouldClone bool

	RepoUrl     string
	Token       string
	Path        string `validate:"required"`
	TrunkBranch string
	RemoteName  string
	User        string
	Email       string
}

// Validate validates exposed fields according to the `validate` tag.
func (g *GitOptions) Validate() error {
	g.User = util.Or(g.User, DefaultGitUser)
	g.Email = util.Or(g.Email, DefaultGitEmail)
	g.TrunkBranch = util.Or(g.TrunkBranch, DefaultTrunkBranch)
	g.RemoteName = util.Or(g.RemoteName, DefaultRemoteName)

	return validator.Validate(g)
}

func (g *GitOptions) ShouldCloneIfNotExist() *GitOptions {
	g.shouldClone = true
	return g
}

func (g *GitOptions) basicAuth() *gogithttp.BasicAuth {
	// TODO integrate with k8s secret
	// ref: https://github.com/fluxcd/source-controller/blob/main/pkg/git/options.go
	token := g.Token
	if token == "" {
		return nil
	}

	user := DefaultAuthUser
	var password string
	if strings.Contains(token, ":") {
		slice := strings.Split(token, ":")
		user = slice[0]
		password = slice[1]
	} else {
		password = token
	}

	return &gogithttp.BasicAuth{
		Username: user,
		Password: password,
	}
}

func (g *GitOptions) signature() *object.Signature {
	return &object.Signature{
		Name:  g.User,
		Email: g.Email,
		When:  time.Now(),
	}
}

type Git struct {
	opts *GitOptions
	repo *extgogit.Repository
}

// NewGit creates Git with a go-git repository.
func NewGit(o *GitOptions) (*Git, error) {
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("validate GitOptions struct: %w", err)
	}
	g := NewGitWithoutRepo(o)
	repo, err := extgogit.PlainOpen(g.opts.Path)
	if err == nil {
		g.repo = repo
		return g, nil
	}
	if !g.opts.shouldClone {
		return nil, errors.WithStack(fmt.Errorf("open git repo %s: %w", g.opts.Path, err))
	}

	if err := os.MkdirAll(g.opts.Path, 0o755); err != nil {
		return nil, errors.WithStack(fmt.Errorf("make dir %s: %w", g.opts.Path, err))
	}
	repo, err = g.Clone()
	if err != nil {
		return nil, err
	}
	g.repo = repo
	return g, nil
}

// NewGitWithoutRepo creates Git without setting up a go-git repository.
func NewGitWithoutRepo(o *GitOptions) *Git {
	return &Git{
		opts: o,
	}
}

// Clone clones remote git repo to the given local path.
func (g *Git) Clone(opts ...CloneOpts) (*extgogit.Repository, error) {
	o := &extgogit.CloneOptions{
		URL:           g.opts.RepoUrl,
		Auth:          g.BasicAuth(),
		RemoteName:    g.opts.RemoteName,
		ReferenceName: plumbing.NewBranchReferenceName(g.opts.TrunkBranch),
		Progress:      os.Stdout,
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}

	repo, err := extgogit.PlainClone(g.opts.Path, false, o)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("git clone: %w", err))
	}
	return repo, nil
}

// CloneOpts enables modification of the go-git CheckoutOptions.
type CloneOpts func(o *extgogit.CloneOptions)

// Options returns internal GitOptions.
func (g *Git) Options() *GitOptions {
	return g.opts
}

// Repo returns internal go-git repository.
func (g *Git) Repo() *extgogit.Repository {
	return g.repo
}

// BasicAuth returns the go-git BasicAuth if git token is provided, otherwise nil.
func (g *Git) BasicAuth() *gogithttp.BasicAuth {
	return g.opts.basicAuth()
}

// Signature returns the go-git Signature using user given User/Email or default one.
func (g *Git) Signature() *object.Signature {
	return g.opts.signature()
}

// Branch returns the current branch name.
func (g *Git) Branch() (string, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("resolve head: %w", err))
	}
	return ref.Name().Short(), nil
}

// Branches returns the all branch names at the local repo.
func (g *Git) Branches() ([]*plumbing.Reference, error) {
	var branches []*plumbing.Reference
	it, err := g.repo.Branches()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = it.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, ref)
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return branches, nil
}

// SetUpstream writes branch remote setting to .git/config.
func (g *Git) SetUpstream(branch string) error {
	b := config.Branch{
		Name:   branch,
		Remote: g.opts.RemoteName,
		Merge:  plumbing.NewBranchReferenceName(branch),
	}
	if err := g.repo.CreateBranch(&b); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Head returns the object.Commit of the current repository head.
func (g *Git) Head() (*object.Commit, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("resolve head: %w", err))
	}
	h := ref.Hash()
	c, err := g.repo.CommitObject(h)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("create commit object: %w", err))
	}
	return c, nil
}

// Checkout switches git branch to the given one and returns git worktree.
func (g *Git) Checkout(opts ...CheckoutOpts) (*extgogit.Worktree, error) {
	w, err := g.repo.Worktree()
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}

	o := &extgogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(g.opts.TrunkBranch),
		// NOTE no option -> merge reset, keep -> soft reset, force -> hard reset
		// Keep:   true,
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}

	ref, _ := g.repo.Head()
	if ref != nil && ref.Name() == o.Branch {
		return w, nil
	}

	if err := w.Checkout(o); err != nil {
		return nil, errors.WithStack(fmt.Errorf("checkout to %s: %w", o.Branch.Short(), err))
	}

	return w, nil
}

// CheckoutOpts enables modification of the go-git CheckoutOptions.
type CheckoutOpts func(o *extgogit.CheckoutOptions)

func CheckoutOptsCreateNew() CheckoutOpts {
	return func(o *extgogit.CheckoutOptions) {
		o.Create = true
	}
}

func CheckoutOptsSoftReset() CheckoutOpts {
	return func(o *extgogit.CheckoutOptions) {
		o.Keep = true
	}
}

func CheckoutOptsTo(branch string) CheckoutOpts {
	return func(o *extgogit.CheckoutOptions) {
		o.Branch = plumbing.NewBranchReferenceName(branch)
	}
}

// Commit execute `git commit -m` with given message.
func (g *Git) Commit(msg string, opts ...CommitOpts) (plumbing.Hash, error) {
	w, err := g.repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}

	o := &extgogit.CommitOptions{
		Author:    g.Signature(),
		Committer: g.Signature(),
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}
	h, err := w.Commit(msg, o)
	if err != nil {
		return plumbing.ZeroHash, errors.WithStack(fmt.Errorf("git commit: %w", err))
	}
	return h, nil
}

// CommitOpts enables modification of the go-git CommitOptions.
type CommitOpts func(o *extgogit.CommitOptions)

// Add executes `git add` for all created/modified/deleted files included in the given path.
func (g *Git) Add(path string) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}

	if _, err := w.Add(path); err != nil {
		return errors.WithStack(fmt.Errorf("git add: %w", err))
	}

	// NOTE go-git does not remove deleted files from index except for specifying deleted files individually.
	// https://github.com/go-git/go-git/issues/113
	stmap, err := w.Status()
	if err != nil {
		return errors.WithStack(fmt.Errorf("get status: %w", err))
	}
	prefix := filepath.Clean(path)
	for fpath, st := range stmap {
		if prefix != "." && !strings.HasPrefix(fpath, prefix) {
			continue
		}
		if st.Worktree != extgogit.Deleted {
			continue
		}
		if _, err := w.Add(fpath); err != nil {
			return errors.WithStack(fmt.Errorf("git add %s: %w", fpath, err))
		}
	}

	return nil
}

// Push pushes the specified git branch to remote. If branch is empty, it pushes the branch set by GitOptions.TrunkBranch.
func (g *Git) Push(opts ...PushOpts) error {
	branch := g.opts.TrunkBranch
	o := &extgogit.PushOptions{
		RemoteName: g.opts.RemoteName,
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(plumbing.NewBranchReferenceName(branch) + ":" + plumbing.NewBranchReferenceName(branch)),
		},
		Auth: g.BasicAuth(),
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}
	if err := g.repo.Push(o); err != nil {
		if !errors.Is(err, extgogit.NoErrAlreadyUpToDate) {
			return errors.WithStack(fmt.Errorf("git push: %w", err))
		}
	}
	return nil
}

// PushOpts enables modification of the go-git PushOptions.
type PushOpts func(o *extgogit.PushOptions)

func PushOptBranch(branch string) PushOpts {
	return func(o *extgogit.PushOptions) {
		o.RefSpecs = []config.RefSpec{
			config.RefSpec(plumbing.NewBranchReferenceName(branch) + ":" + plumbing.NewBranchReferenceName(branch)),
		}
	}
}

// Pull pulls the specified git branch from remote to local.
func (g *Git) Pull(opts ...PullOpts) error {
	o := &extgogit.PullOptions{
		RemoteName:   g.opts.RemoteName,
		SingleBranch: false,
		Progress:     os.Stdout,
		Auth:         g.BasicAuth(),
	}
	// NOTE explicit head resolution is needed since go-git ReferenceName default does not work.
	if ref, err := g.repo.Head(); err == nil {
		o.ReferenceName = ref.Name()
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}
	if err := w.Pull(o); err != nil {
		if !errors.Is(err, extgogit.NoErrAlreadyUpToDate) {
			return errors.WithStack(fmt.Errorf("git pull: %w", err))
		}
	}
	return nil
}

// PullOpts enables modification of the go-git PullOptions.
type PullOpts func(o *extgogit.PullOptions)

func PullOptsReference(name plumbing.ReferenceName) PullOpts {
	return func(o *extgogit.PullOptions) {
		o.ReferenceName = name
	}
}

// ResetOpts enables modification of the go-git ResetOptions.
type ResetOpts func(o *extgogit.ResetOptions)

func ResetOptsHard() ResetOpts {
	return func(o *extgogit.ResetOptions) {
		o.Mode = extgogit.HardReset
	}
}

// Reset runs git-reset with supplied options.
func (g *Git) Reset(opts ...ResetOpts) error {
	o := &extgogit.ResetOptions{}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}
	if err := w.Reset(o); err != nil {
		return errors.WithStack(fmt.Errorf("run git-reset: %w", err))
	}
	return nil
}

// RemoveBranch removes the local branch.
func (g *Git) RemoveBranch(rn plumbing.ReferenceName) error {
	err := g.repo.Storer.RemoveReference(rn)
	return errors.WithStack(err)
}

// RemoveGoneBranches removes all gone branches.
func (g *Git) RemoveGoneBranches() error {
	remote, err := g.Remote("")
	if err != nil {
		return err
	}
	remoteBrs, err := remote.Branches()
	if err != nil {
		return err
	}
	nameSet := util.NewSet[string]()
	for _, br := range remoteBrs {
		nameSet.Add(br.Name().Short())
	}

	localBrs, err := g.Branches()
	if err != nil {
		return err
	}
	for _, br := range localBrs {
		if !nameSet.Has(br.Name().Short()) {
			err = multierr.Append(err, g.RemoveBranch(br.Name()))
		}
	}
	if err != nil {
		return util.JoinErr("remote gone branches:", err)
	}

	return nil
}

// Remote returns GitRemote of the supplied remote name. If not supplied, it returns the one of GitOptions.RemoteName.
func (g *Git) Remote(name string) (*GitRemote, error) {
	if name == "" {
		name = g.opts.RemoteName
	}
	r, err := g.repo.Remote(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	remote := &GitRemote{
		opts:   g.opts,
		remote: r,
	}
	return remote, nil
}

type GitRemote struct {
	opts   *GitOptions
	remote *extgogit.Remote
}

func (r *GitRemote) Name() string {
	c := r.remote.Config()
	return c.Name
}

// BasicAuth returns the go-git BasicAuth if git token is provided, otherwise nil.
func (r *GitRemote) BasicAuth() *gogithttp.BasicAuth {
	return r.opts.basicAuth()
}

// ListOpts enables modification of the go-git ListOptions.
type ListOpts func(o *extgogit.ListOptions)

// Branches lists the branches of the remote repository.
func (r *GitRemote) Branches(opts ...ListOpts) ([]*plumbing.Reference, error) {
	o := &extgogit.ListOptions{
		Auth: r.BasicAuth(),
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}
	refs, err := r.remote.List(o)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var branches []*plumbing.Reference
	for _, ref := range refs {
		if ref.Name().IsBranch() {
			branches = append(branches, ref)
		}
	}

	return branches, nil
}

// RemoveBranch removes the remote branch.
func (r *GitRemote) RemoveBranch(rn plumbing.ReferenceName, opts ...PushOpts) error {
	o := &extgogit.PushOptions{
		RemoteName: r.opts.RemoteName,
		RefSpecs: []config.RefSpec{
			config.RefSpec(":" + rn.String()),
		},
		Auth: r.BasicAuth(),
	}
	for _, tr := range opts {
		if tr != nil {
			tr(o)
		}
	}
	err := r.remote.Push(o)
	if err != nil && !errors.Is(err, extgogit.NoErrAlreadyUpToDate) {
		return errors.WithStack(err)
	}
	return nil
}

// IsTrackedAndChanged returns true if git file status is neither untracked nor unmodified.
func IsTrackedAndChanged(c extgogit.StatusCode) bool {
	return c != extgogit.Untracked && c != extgogit.Unmodified
}

// IsBothWorktreeAndStagingTrackedAndChanged returns true if both stating and worktree statuses
// of the given file are tracked and changed.
func IsBothWorktreeAndStagingTrackedAndChanged(st extgogit.FileStatus) bool {
	return IsTrackedAndChanged(st.Worktree) && IsTrackedAndChanged(st.Staging)
}

// IsEitherWorktreeOrStagingTrackedAndChanged returns true if either stating or worktree status
// of the given file is tracked and changed.
func IsEitherWorktreeOrStagingTrackedAndChanged(st extgogit.FileStatus) bool {
	return IsTrackedAndChanged(st.Worktree) || IsTrackedAndChanged(st.Staging)
}
