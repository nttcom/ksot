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

package githelper

import (
	"os"
	"testing"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
)

func InitRepo(t *testing.T, branch string) (*extgogit.Repository, string) {
	dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, false)
	testhelper.ExitOnErr(t, err)

	testhelper.ExitOnErr(t, CreateFileWithAdding(repo, "README.md", "# test"))
	_, err = Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)
	testhelper.ExitOnErr(t, CreateBranch(repo, branch))
	return repo, dir
}

func InitBareRepo(t *testing.T) (*extgogit.Repository, string) {
	dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, true)
	testhelper.ExitOnErr(t, err)
	return repo, dir
}

func InitRepoWithRemote(t *testing.T, branch, remote string) (*extgogit.Repository, string, string) {
	_, dirBare := InitBareRepo(t)
	repo, dir := InitRepo(t, branch)
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: remote,
		URLs: []string{dirBare},
	})
	testhelper.ExitOnErr(t, err)

	return repo, dir, dirBare
}

func SetupRemoteRepo(t *testing.T, opt *gogit.GitOptions) (*gogit.GitRemote, *gogit.Git, string) {
	_, dir, _ := InitRepoWithRemote(t, "main", "origin")

	opt.Path = dir
	git, err := gogit.NewGit(opt)
	testhelper.ExitOnErr(t, err)

	remote, err := git.Remote("origin")
	testhelper.ExitOnErr(t, err)
	return remote, git, dir
}

func CloneRepo(t *testing.T, opts *extgogit.CloneOptions) (*extgogit.Repository, string) {
	dir := t.TempDir()
	repo, err := extgogit.PlainClone(dir, false, opts)
	testhelper.ExitOnErr(t, err)
	return repo, dir
}

func CreateFile(wt *extgogit.Worktree, path, content string) error {
	f, err := wt.Filesystem.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write([]byte(content)); err != nil {
		return err
	}
	return nil
}

func ModifyFile(wt *extgogit.Worktree, path, content string) error {
	f, err := wt.Filesystem.OpenFile(path, os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write([]byte(content)); err != nil {
		return err
	}
	return nil
}

func DeleteFile(wt *extgogit.Worktree, path string) error {
	if err := wt.Filesystem.Remove(path); err != nil {
		return err
	}
	return nil
}

func CreateFileWithAdding(repo *extgogit.Repository, path, content string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := CreateFile(wt, path, content); err != nil {
		return err
	}
	if _, err = wt.Add(path); err != nil {
		return err
	}
	return nil
}

func DeleteFileWithAdding(repo *extgogit.Repository, path string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if _, err = wt.Remove(path); err != nil {
		return err
	}
	return nil
}

func Commit(repo *extgogit.Repository, time time.Time) (plumbing.Hash, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return plumbing.Hash{}, err
	}
	return wt.Commit("Updated", &extgogit.CommitOptions{
		Author:    MockSignature(time),
		Committer: MockSignature(time),
	})
}

func Push(repo *extgogit.Repository, branch, remote string) error {
	o := &extgogit.PushOptions{
		RemoteName: remote,
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(plumbing.NewBranchReferenceName(branch) + ":" + plumbing.NewBranchReferenceName(branch)),
		},
	}
	return repo.Push(o)
}

func CreateBranch(repo *extgogit.Repository, branch string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	h, err := repo.Head()
	if err != nil {
		return err
	}
	return wt.Checkout(&extgogit.CheckoutOptions{
		Hash:   h.Hash(),
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
		Create: true,
		Keep:   true,
	})
}

func MockSignature(time time.Time) *object.Signature {
	return &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time,
	}
}

func GetStatus(t *testing.T, repo *extgogit.Repository) extgogit.Status {
	w, err := repo.Worktree()
	testhelper.ExitOnErr(t, err)
	stmap, err := w.Status()
	testhelper.ExitOnErr(t, err)

	return stmap
}

func GetBranch(t *testing.T, repo *extgogit.Repository) string {
	ref, err := repo.Head()
	testhelper.ExitOnErr(t, err)

	return ref.Name().Short()
}

func GetRemoteBranches(t *testing.T, repo *extgogit.Repository, remoteName string) []*plumbing.Reference {
	remote, err := repo.Remote(remoteName)
	testhelper.ExitOnErr(t, err)
	branches, err := remote.List(&extgogit.ListOptions{})
	testhelper.ExitOnErr(t, err)

	return branches
}

func GetRemoteBranch(t *testing.T, repo *extgogit.Repository, remoteName, branchName string) *plumbing.Reference {
	branches := GetRemoteBranches(t, repo, remoteName)
	for _, ref := range branches {
		if ref.Name().Short() == branchName {
			return ref
		}
	}
	return nil
}
