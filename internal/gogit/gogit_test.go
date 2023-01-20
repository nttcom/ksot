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

package gogit_test

import (
	"testing"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/internal/testing/githelper"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestGitOptions_Validate(t *testing.T) {
	newValidStruct := func(t func(git *gogit.GitOptions)) *gogit.GitOptions {
		g := &gogit.GitOptions{
			Path: "./",
		}
		t(g)
		return g
	}

	tests := []struct {
		name      string
		transform func(g *gogit.GitOptions)
		wantErr   bool
	}{
		{
			"ok",
			func(g *gogit.GitOptions) {},
			false,
		},
		{
			"err: path is empty",
			func(g *gogit.GitOptions) {
				g.Path = ""
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newValidStruct(tt.transform)
			err := g.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestNewGit(t *testing.T) {
	t.Run("ok: use existing", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		opt := &gogit.GitOptions{
			Path: dir,
		}
		testhelper.ExitOnErr(t, opt.Validate())

		g, err := gogit.NewGit(opt)
		assert.Nil(t, err)
		assert.Equal(t, opt, g.Options())

		want, _ := repo.Head()
		got, _ := g.Repo().Head()
		assert.Equal(t, want, got)
	})

	t.Run("ok: clone", func(t *testing.T) {
		repoPusher, dir, remoteUrl := githelper.InitRepoWithRemote(t, "main", "origin")
		testhelper.ExitOnErr(t, githelper.Push(repoPusher, "main", "origin"))

		opt := &gogit.GitOptions{
			RepoUrl: remoteUrl,
			Path:    dir,
		}
		testhelper.ExitOnErr(t, opt.Validate())
		opt.ShouldCloneIfNotExist()

		g, err := gogit.NewGit(opt)
		assert.Nil(t, err)
		assert.Equal(t, opt, g.Options())

		want, _ := repoPusher.Head()
		got, _ := g.Repo().Head()
		assert.Equal(t, want, got)
	})

	t.Run("err: no repo without shouldClone flag", func(t *testing.T) {
		repoPusher, _, remoteUrl := githelper.InitRepoWithRemote(t, "main", "origin")
		testhelper.ExitOnErr(t, githelper.Push(repoPusher, "main", "origin"))

		dir := t.TempDir()
		opt := &gogit.GitOptions{
			RepoUrl: remoteUrl,
			Path:    dir,
		}
		testhelper.ExitOnErr(t, opt.Validate())

		_, err := gogit.NewGit(opt)
		assert.Error(t, err)
	})
}

func TestGit_Clone(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repoPusher, _, remoteUrl := githelper.InitRepoWithRemote(t, "main", "origin")
		testhelper.ExitOnErr(t, githelper.Push(repoPusher, "main", "origin"))

		dir := t.TempDir()
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
			RepoUrl: remoteUrl,
			Path:    dir,
		})
		testhelper.ExitOnErr(t, g.Options().Validate())

		_, err := g.Clone()
		assert.Nil(t, err)
	})

	t.Run("err: url not given", func(t *testing.T) {
		dir := t.TempDir()
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, g.Options().Validate())

		_, err := g.Clone()
		assert.Error(t, err)
	})
}

func TestGit_BasicAuth(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  *githttp.BasicAuth
	}{
		{
			"both username and password",
			"user:pass",
			&githttp.BasicAuth{
				Username: "user",
				Password: "pass",
			},
		},
		{
			"only password",
			"pass",
			&githttp.BasicAuth{
				Username: "anonymous",
				Password: "pass",
			},
		},
		{
			"not set",
			"",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
				Token: tt.token,
			})
			assert.Equal(t, tt.want, g.BasicAuth())
		})
	}
}

func TestGit_Signature(t *testing.T) {
	t.Run("given user/email", func(t *testing.T) {
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
			User:  "test-user",
			Email: "test-email",
		})
		got := g.Signature()
		assert.Equal(t, "test-user", got.Name)
		assert.Equal(t, "test-email", got.Email)
	})
	t.Run("default", func(t *testing.T) {
		o := &gogit.GitOptions{Path: "dummy"}
		testhelper.ExitOnErr(t, o.Validate())
		g := gogit.NewGitWithoutRepo(o)
		got := g.Signature()
		assert.Equal(t, gogit.DefaultGitUser, got.Name)
		assert.Equal(t, gogit.DefaultGitEmail, got.Email)
	})
}

func TestGit_Head(t *testing.T) {
	repo, dir := githelper.InitRepo(t, "main")
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "test", "hash"))
	want, err := githelper.Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)

	g, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	testhelper.ExitOnErr(t, err)

	c, err := g.Head()
	testhelper.ExitOnErr(t, err)
	assert.Equal(t, want, c.Hash)
}

func TestGit_Checkout(t *testing.T) {
	t.Run("ok: checkout to main", func(t *testing.T) {
		_, dir := githelper.InitRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)

		_, err = g.Checkout()
		testhelper.ExitOnErr(t, err)

		b, err := g.Branch()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, "main", b)
	})

	t.Run("ok: checkout to specified trunk", func(t *testing.T) {
		branchName := "test-branch"
		_, dir := githelper.InitRepo(t, branchName)
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			TrunkBranch: branchName,
		})
		testhelper.ExitOnErr(t, err)

		_, err = g.Checkout()
		testhelper.ExitOnErr(t, err)

		b, err := g.Branch()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, branchName, b)
	})

	t.Run("ok: checkout to new branch", func(t *testing.T) {
		_, dir := githelper.InitRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)

		b, err := g.Branch()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, "test", b)
	})

	t.Run("err: checkout to existing branch with create opt", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, githelper.CreateBranch(repo, "test"))

		_, err = g.Checkout(gogit.CheckoutOptsTo("main"), gogit.CheckoutOptsCreateNew())
		assert.Error(t, err)
	})

	t.Run("err: checkout to new branch without create opt", func(t *testing.T) {
		_, dir := githelper.InitRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"))
		assert.Error(t, err)
	})
}

func TestGit_Commit(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "test", "dummy"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)
		_, err = g.Commit("added: test")
		assert.Nil(t, err)
	})

	t.Run("ok: other trunk branch", func(t *testing.T) {
		testTrunk := "test-branch"
		repo, dir := githelper.InitRepo(t, testTrunk)
		testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "test", "dummy"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			TrunkBranch: testTrunk,
		})
		testhelper.ExitOnErr(t, err)
		h, err := g.Commit("added: test")
		assert.Nil(t, err)

		b, err := g.Branch()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, testTrunk, b)

		c, err := g.Head()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, h, c.Hash)
	})

	t.Run("ok: commit even when no change", func(t *testing.T) {
		_, dir := githelper.InitRepo(t, "main")

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)
		_, err = g.Commit("no change")
		assert.Nil(t, err)
	})
}

func TestGit_Add(t *testing.T) {
	t.Run("ok: create new", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		wt, err := repo.Worktree()
		testhelper.ExitOnErr(t, err)

		filepath := "test/added.txt"
		testhelper.ExitOnErr(t, githelper.CreateFile(wt, filepath, "dummy"))
		testhelper.ExitOnErr(t, githelper.ModifyFile(wt, "README.md", "foobar"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)
		err = g.Add("test")
		assert.Nil(t, err)

		stmap, err := wt.Status()
		testhelper.ExitOnErr(t, err)
		count := 0
		for fpath, st := range stmap {
			if st.Staging == extgogit.Unmodified {
				continue
			}
			count += 1
			assert.Equal(t, fpath, filepath)
			assert.Equal(t, st.Staging, extgogit.Added)
		}
		assert.Equal(t, count, 1)
	})

	t.Run("ok: modify existing", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		wt, err := repo.Worktree()
		testhelper.ExitOnErr(t, err)

		testhelper.ExitOnErr(t, githelper.ModifyFile(wt, "README.md", "foobar"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)
		err = g.Add("")
		assert.Nil(t, err)

		stmap, err := wt.Status()
		testhelper.ExitOnErr(t, err)
		count := 0
		for fpath, st := range stmap {
			if st.Staging == extgogit.Unmodified {
				continue
			}
			count += 1
			assert.Equal(t, fpath, "README.md")
			assert.Equal(t, st.Staging, extgogit.Modified)
		}
		assert.Equal(t, count, 1)
	})

	t.Run("ok: delete existing", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		wt, err := repo.Worktree()
		testhelper.ExitOnErr(t, err)

		t.Log("#################", dir)
		testhelper.ExitOnErr(t, githelper.DeleteFile(wt, "README.md"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)
		err = g.Add("")
		assert.Nil(t, err)

		stmap, err := wt.Status()
		testhelper.ExitOnErr(t, err)
		count := 0
		for fpath, st := range stmap {
			if st.Staging == extgogit.Unmodified {
				continue
			}
			count += 1
			assert.Equal(t, fpath, "README.md")
			assert.Equal(t, st.Staging, extgogit.Deleted)
		}
		assert.Equal(t, count, 1)
	})
}

func TestGit_Push(t *testing.T) {
	remoteRepo, url := githelper.InitBareRepo(t)

	t.Run("ok", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		testRemote := "test-remote"
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: testRemote,
			URLs: []string{url},
		})
		testhelper.ExitOnErr(t, err)

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: testRemote,
		})
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "test", "push"))
		wantMsg := "git commit which should be pushed to remote"
		_, err = g.Commit(wantMsg)
		testhelper.ExitOnErr(t, err)

		err = g.Push(gogit.PushOptBranch("main"))
		testhelper.ExitOnErr(t, err)

		ref, err := remoteRepo.Reference(plumbing.NewBranchReferenceName("main"), false)
		testhelper.ExitOnErr(t, err)
		c, err := repo.CommitObject(ref.Hash())
		testhelper.ExitOnErr(t, err)

		assert.Equal(t, wantMsg, c.Message)
	})

	t.Run("err: remote not exist", func(t *testing.T) {
		repo, dir := githelper.InitRepo(t, "main")
		noExistRemote := "not-exist"

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			RemoteName:  noExistRemote,
			TrunkBranch: "main",
		})
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "test", "push"))
		_, err = g.Commit("added: test")
		testhelper.ExitOnErr(t, err)

		err = g.Push()
		assert.Error(t, err)
	})
}

func TestGit_SetUpstream(t *testing.T) {
	_, dirBare := githelper.InitBareRepo(t)
	testRemote := "test-remote"

	repo, dir := githelper.InitRepo(t, "main")
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: testRemote,
		URLs: []string{dirBare},
	})
	testhelper.ExitOnErr(t, err)
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path:       dir,
		RemoteName: testRemote,
	})
	testhelper.ExitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
	testhelper.ExitOnErr(t, err)

	err = git.SetUpstream("test")
	assert.Nil(t, err)

	c, err := repo.Storer.Config()
	testhelper.ExitOnErr(t, err)

	exists := false
	for name, r := range c.Branches {
		if name == "test" {
			assert.Equal(t, testRemote, r.Remote)
			assert.Equal(t, plumbing.NewBranchReferenceName("test"), r.Merge)
			exists = true
		}
	}
	assert.True(t, exists)
}

func TestGit_Branches(t *testing.T) {
	_, dir := githelper.InitRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	testhelper.ExitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
	testhelper.ExitOnErr(t, err)

	refs, err := git.Branches()
	assert.Nil(t, err)
	assert.Len(t, refs, 3)
	for _, ref := range refs {
		assert.Contains(t, []string{"main", "test", "master"}, ref.Name().Short())
	}
}

func TestGit_Pull(t *testing.T) {
	testRemote := "test-remote"

	setup := func(t *testing.T, beforeCloneFn func(*gogit.Git)) (*gogit.Git, *gogit.Git) {
		// setup remote bare repo
		_, dirBare := githelper.InitBareRepo(t)

		// setup pusher
		repoPusher, dirPusher := githelper.InitRepo(t, "main")
		_, err := repoPusher.CreateRemote(&config.RemoteConfig{
			Name: testRemote,
			URLs: []string{dirBare},
		})
		testhelper.ExitOnErr(t, err)
		gitPusher, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dirPusher,
			RemoteName: testRemote,
		})
		testhelper.ExitOnErr(t, err)

		_, err = gitPusher.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)

		if beforeCloneFn != nil {
			beforeCloneFn(gitPusher)
		}

		// setup puller by git clone
		_, dirPuller := githelper.CloneRepo(t, &extgogit.CloneOptions{
			URL:        dirBare,
			RemoteName: testRemote,
		})
		gitPuller, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dirPuller,
			RemoteName: testRemote,
		})
		testhelper.ExitOnErr(t, err)

		return gitPusher, gitPuller
	}

	t.Run("ok", func(t *testing.T) {
		gitPusher, gitPuller := setup(t, func(pusher *gogit.Git) {
			testhelper.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("master")))
			testhelper.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("test")))
		})

		// push branch
		testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(gitPuller.Repo(), "test", "push"))
		wantMsg := "git commit which should be pushed to remote"
		want, err := gitPusher.Commit(wantMsg)
		testhelper.ExitOnErr(t, err)

		testhelper.ExitOnErr(t, gitPusher.Push(gogit.PushOptBranch("test")))

		// pull branch
		_, err = gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)

		err = gitPuller.Pull()
		testhelper.ExitOnErr(t, err)

		got, err := gitPuller.Head()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, want.String(), got.Hash.String())
	})

	t.Run("ok: no update", func(t *testing.T) {
		gitPusher, gitPuller := setup(t, func(pusher *gogit.Git) {
			testhelper.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("master")))
			testhelper.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("test")))
		})

		head, err := gitPusher.Head()
		testhelper.ExitOnErr(t, err)
		want := head.Hash

		// pull branch
		_, err = gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)

		err = gitPuller.Pull()
		testhelper.ExitOnErr(t, err)

		got, err := gitPuller.Head()
		testhelper.ExitOnErr(t, err)
		assert.Equal(t, want.String(), got.Hash.String())
	})

	t.Run("err: upstream branch not exist", func(t *testing.T) {
		_, gitPuller := setup(t, func(g *gogit.Git) {
			testhelper.ExitOnErr(t, g.Push(gogit.PushOptBranch("master")))
		})

		_, err := gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)

		err = gitPuller.Pull()
		assert.Error(t, err)
	})

	t.Run("err: remote repo not exist", func(t *testing.T) {
		_, dir := githelper.InitRepo(t, "main")
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: testRemote,
		})
		testhelper.ExitOnErr(t, err)

		err = git.Pull()
		assert.Error(t, err)
	})
}

func TestGit_Reset(t *testing.T) {
	repo, dir := githelper.InitRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	testhelper.ExitOnErr(t, err)
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "test", "hash"))

	w, err := repo.Worktree()
	testhelper.ExitOnErr(t, err)

	st, err := w.Status()
	testhelper.ExitOnErr(t, err)
	assert.Greater(t, len(st), 0)

	err = git.Reset(gogit.ResetOptsHard())
	assert.Nil(t, err)

	st, err = w.Status()
	testhelper.ExitOnErr(t, err)
	assert.Equal(t, len(st), 0)
}

func TestGit_RemoveBranch(t *testing.T) {
	_, dir := githelper.InitRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	testhelper.ExitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("foo"), gogit.CheckoutOptsCreateNew())
	testhelper.ExitOnErr(t, err)
	refs, err := git.Branches()
	testhelper.ExitOnErr(t, err)
	assert.Len(t, refs, 3)

	testhelper.ExitOnErr(t, git.RemoveBranch(plumbing.NewBranchReferenceName("foo")))

	refs, err = git.Branches()
	testhelper.ExitOnErr(t, err)
	assert.Len(t, refs, 2)
}

func TestGit_RemoveGoneBranches(t *testing.T) {
	repo, dir, _ := githelper.InitRepoWithRemote(t, "main", "origin")
	testhelper.ExitOnErr(t, githelper.Push(repo, "main", "origin"))

	testhelper.ExitOnErr(t, githelper.CreateBranch(repo, "foo"))
	testhelper.ExitOnErr(t, githelper.Push(repo, "foo", "origin"))

	testhelper.ExitOnErr(t, githelper.CreateBranch(repo, "bar"))
	testhelper.ExitOnErr(t, githelper.Push(repo, "bar", "origin"))

	g, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	testhelper.ExitOnErr(t, err)

	refs, err := g.Branches()
	testhelper.ExitOnErr(t, err)
	assert.Len(t, refs, 4)

	remote, err := g.Remote("origin")
	testhelper.ExitOnErr(t, err)
	testhelper.ExitOnErr(t, remote.RemoveBranch(plumbing.NewBranchReferenceName("bar")))

	err = g.RemoveGoneBranches()
	assert.Nil(t, err)

	refs, err = g.Branches()
	testhelper.ExitOnErr(t, err)
	assert.Len(t, refs, 2)
}

func TestGit_Branch(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		_, dirBare := githelper.InitBareRepo(t)
		repo, dir := githelper.InitRepo(t, "main")
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{dirBare},
		})
		testhelper.ExitOnErr(t, err)
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		testhelper.ExitOnErr(t, err)

		remote, err := git.Remote("origin")
		assert.Nil(t, err)
		assert.Equal(t, "origin", remote.Name())
	})

	t.Run("ok: use default", func(t *testing.T) {
		_, dirBare := githelper.InitBareRepo(t)
		repo, dir := githelper.InitRepo(t, "main")
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{dirBare},
		})
		testhelper.ExitOnErr(t, err)
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: "origin",
		})
		testhelper.ExitOnErr(t, err)

		remote, err := git.Remote("")
		assert.Nil(t, err)
		assert.Equal(t, "origin", remote.Name())
	})

	t.Run("err: remote not exist", func(t *testing.T) {
		_, dir := githelper.InitRepo(t, "main")
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: "origin",
		})
		testhelper.ExitOnErr(t, err)

		_, err = git.Remote("origin")
		assert.Error(t, err)
	})
}

func TestGitBranch_BasicAuth(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  *githttp.BasicAuth
	}{
		{
			"both username and password",
			"user:pass",
			&githttp.BasicAuth{
				Username: "user",
				Password: "pass",
			},
		},
		{
			"only password",
			"pass",
			&githttp.BasicAuth{
				Username: "anonymous",
				Password: "pass",
			},
		},
		{
			"not set",
			"",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &gogit.GitOptions{
				Token: tt.token,
			}
			remote, _, _ := githelper.SetupRemoteRepo(t, opt)
			assert.Equal(t, tt.want, remote.BasicAuth())
		})
	}
}

func TestGitRemote_Branches(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		remote, git, _ := githelper.SetupRemoteRepo(t, &gogit.GitOptions{})

		_, err := git.Checkout(gogit.CheckoutOptsTo("foo"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, git.Push(gogit.PushOptBranch("foo")))

		_, err = git.Checkout(gogit.CheckoutOptsTo("bar"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, git.Push(gogit.PushOptBranch("bar")))

		branches, err := remote.Branches()
		assert.Nil(t, err)
		assert.Len(t, branches, 2)
		for _, b := range branches {
			assert.Contains(t, []string{"refs/heads/foo", "refs/heads/bar"}, b.Name().String())
		}
	})

	t.Run("err: no branch", func(t *testing.T) {
		remote, _, _ := githelper.SetupRemoteRepo(t, &gogit.GitOptions{})
		_, err := remote.Branches()
		assert.Error(t, err)
	})
}

func TestGitRemote_RemoveBranch(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		remote, git, _ := githelper.SetupRemoteRepo(t, &gogit.GitOptions{})

		_, err := git.Checkout(gogit.CheckoutOptsTo("foo"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, git.Push(gogit.PushOptBranch("foo")))

		_, err = git.Checkout(gogit.CheckoutOptsTo("bar"), gogit.CheckoutOptsCreateNew())
		testhelper.ExitOnErr(t, err)
		testhelper.ExitOnErr(t, git.Push(gogit.PushOptBranch("bar")))

		err = remote.RemoveBranch(plumbing.NewBranchReferenceName("foo"))
		assert.Nil(t, err)

		branches, err := remote.Branches()
		testhelper.ExitOnErr(t, err)
		assert.Len(t, branches, 1)
		assert.Equal(t, "refs/heads/bar", branches[0].Name().String())
	})

	t.Run("ok: branch not found", func(t *testing.T) {
		remote, _, _ := githelper.SetupRemoteRepo(t, &gogit.GitOptions{})
		err := remote.RemoveBranch(plumbing.NewBranchReferenceName("foo"))
		assert.Nil(t, err)
	})
}

func TestIsTrackedAndChanged(t *testing.T) {
	tests := []struct {
		given extgogit.StatusCode
		want  bool
	}{
		{extgogit.Unmodified, false},
		{extgogit.Untracked, false},
		{extgogit.Modified, true},
		{extgogit.Added, true},
		{extgogit.Deleted, true},
		{extgogit.Renamed, true},
		{extgogit.Copied, true},
		{extgogit.UpdatedButUnmerged, true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, gogit.IsTrackedAndChanged(tt.given))
	}
}

func TestIsBothWorktreeAndStagingTrackedAndChanged(t *testing.T) {
	tests := []struct {
		given extgogit.FileStatus
		want  bool
	}{
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified}, true},
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified}, false},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified}, false},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified}, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, gogit.IsBothWorktreeAndStagingTrackedAndChanged(tt.given))
	}
}

func TestIsEitherWorktreeOrStagingTrackedAndChanged(t *testing.T) {
	tests := []struct {
		given extgogit.FileStatus
		want  bool
	}{
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified}, true},
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified}, true},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified}, true},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified}, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, gogit.IsEitherWorktreeOrStagingTrackedAndChanged(tt.given))
	}
}
