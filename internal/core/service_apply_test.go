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

package core_test

import (
	"regexp"
	"testing"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/internal/testing/githelper"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestServiceApplyCfg_Validate(t *testing.T) {
	newValidStruct := func(t func(cfg *core.ServiceApplyCfg)) *core.ServiceApplyCfg {
		cfg := &core.ServiceApplyCfg{
			RootCfg: core.RootCfg{
				ConfigRootPath: "./",
			},
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *core.ServiceApplyCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *core.ServiceApplyCfg) {},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCheckGitStatus(t *testing.T) {
	tests := []struct {
		st      extgogit.Status
		wantErr int
	}{
		{
			extgogit.Status{
				"computed/device1.cue":           &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
				"devices/device1/config.cue":     &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
				"services/foo/one/two/input.cue": &extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
			},
			0,
		},
		{
			extgogit.Status{
				"computed/device1.cue":           &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
				"devices/device1/config.cue":     &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
				"services/foo/one/two/input.cue": &extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified},
			},
			3,
		},
	}

	for _, tt := range tests {
		err := core.CheckGitStatus(tt.st)
		if tt.wantErr > 0 {
			reg := regexp.MustCompile(".cue")
			assert.Equal(t, tt.wantErr, len(reg.FindAllString(err.Error(), -1)))
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestCheckGitFileStatus(t *testing.T) {
	tests := []struct {
		path    string
		st      extgogit.FileStatus
		wantErr bool
	}{
		{
			"computed/device1.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			true,
		},
		{
			"computed/device1.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"devices/device1/config.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			true,
		},
		{
			"devices/device1/config.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified},
			true,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			false,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.UpdatedButUnmerged},
			true,
		},
	}

	for _, tt := range tests {
		err := core.CheckGitFileStatus(tt.path, tt.st)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestServiceCompilePlan_Do(t *testing.T) {
	var err error
	repo, dir := githelper.InitRepo(t, "main")
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/one/input.cue", "{}"))
	_, err = githelper.Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)

	testhelper.ExitOnErr(t, githelper.DeleteFileWithAdding(repo, "services/foo/one/input.cue"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/two/input.cue", "{}"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/three/input.cue", "{}"))

	stmap := githelper.GetStatus(t, repo)
	plan := core.NewServiceCompilePlan(stmap, dir)
	updated := 0
	deleted := 0
	err = plan.Do(context.Background(),
		func(ctx context.Context, sp kuesta.ServicePath) error {
			deleted++
			assert.Equal(t, "foo", sp.Service)
			assert.Contains(t, []string{"one"}, sp.Keys[0])
			return nil
		},
		func(ctx context.Context, sp kuesta.ServicePath) error {
			updated++
			assert.Equal(t, "foo", sp.Service)
			assert.Contains(t, []string{"two", "three"}, sp.Keys[0])
			return nil
		})
	assert.Equal(t, 1, deleted)
	assert.Equal(t, 2, updated)
	assert.Nil(t, err)
}

func TestServiceCompilePlan_IsEmpty(t *testing.T) {
	var err error
	repo, dir := githelper.InitRepo(t, "main")
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/one/input.cue", "{}"))
	_, err = githelper.Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)

	stmap := githelper.GetStatus(t, repo)
	plan := core.NewServiceCompilePlan(stmap, dir)
	assert.True(t, plan.IsEmpty())
}

func TestDeviceCompositePlan_Do(t *testing.T) {
	var err error
	repo, dir := githelper.InitRepo(t, "main")
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/one/computed/device1.cue", "{}"))
	_, err = githelper.Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)

	testhelper.ExitOnErr(t, githelper.DeleteFileWithAdding(repo, "services/foo/one/computed/device1.cue"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/two/computed/device2.cue", "{}"))
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/three/computed/device3.cue", "{}"))

	stmap := githelper.GetStatus(t, repo)
	plan := core.NewDeviceCompositePlan(stmap, dir)
	executed := 0
	err = plan.Do(context.Background(),
		func(ctx context.Context, dp kuesta.DevicePath) error {
			executed++
			assert.Contains(t, []string{"device1", "device2", "device3"}, dp.Device)
			return nil
		})
	assert.Equal(t, 3, executed)
	assert.Nil(t, err)
}

func TestDeviceCompositePlan_IsEmpty(t *testing.T) {
	var err error
	repo, dir := githelper.InitRepo(t, "main")
	testhelper.ExitOnErr(t, githelper.CreateFileWithAdding(repo, "services/foo/one/computed/device1.cue", "{}"))
	_, err = githelper.Commit(repo, time.Now())
	testhelper.ExitOnErr(t, err)

	stmap := githelper.GetStatus(t, repo)
	plan := core.NewDeviceCompositePlan(stmap, dir)
	assert.True(t, plan.IsEmpty())
}
