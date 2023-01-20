/*
 Copyright (c) 2023 NTT Communications Corporation

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
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/nttcom/kuesta/internal/testing/githelper"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
)

func SetupGitRepoWithRemote(t *testing.T, remote string) (*git.Repository, string, string) {
	repo, dir, url := githelper.InitRepoWithRemote(t, "main", remote)

	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "devices/device1/actual_config.cue", "{will_deleted: _}"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "devices/device2/actual_config.cue", "{will_updated: _}"))
	_, err := githelper.Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)

	testhelper.ExitOnErr(t, repo.CreateBranch(&config.Branch{
		Name:   "main",
		Remote: remote,
		Merge:  plumbing.NewBranchReferenceName("main"),
	}))

	testhelper.ExitOnErr(t, githelper.Push(repo, "main", remote))

	testhelper.ExitOnErr(t, githelper.DeleteFileWithAdding(repo, "devices/device1/actual_config.cue"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "devices/device2/actual_config.cue", "{updated: _}"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "devices/device3/actual_config.cue", "{added: _}"))

	return repo, dir, url
}
