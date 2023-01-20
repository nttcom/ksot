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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/internal/testing/githelper"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestDecodeSaveConfigRequest(t *testing.T) {
	config := "foobar"

	tests := []struct {
		name    string
		given   string
		want    *core.SaveConfigRequest
		wantErr bool
	}{
		{
			"ok",
			`{"device": "device1", "config": "foobar"}`,
			&core.SaveConfigRequest{
				Device: "device1",
				Config: &config,
			},
			false,
		},
		{
			"err: no device",
			`{"config": "foobar"}`,
			nil,
			true,
		},
		{
			"err: no config",
			`{"device": "device1"}`,
			nil,
			true,
		},
		{
			"err: invalid format",
			`{"device": "device1", "config": "foobar"`,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.given)
			got, err := core.DecodeSaveConfigRequest(r)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMakeSyncCommitMessage(t *testing.T) {
	stmap := extgogit.Status{
		"devices/dvc1/actual_config.cue": &extgogit.FileStatus{Staging: extgogit.Added},
		"devices/dvc2/actual_config.cue": &extgogit.FileStatus{Staging: extgogit.Deleted},
		"devices/dvc3/actual_config.cue": &extgogit.FileStatus{Staging: extgogit.Modified},
	}
	want := `Updated: dvc1 dvc2 dvc3

Devices:
	added:     dvc1
	deleted:   dvc2
	modified:  dvc3`

	assert.Equal(t, want, core.MakeSyncCommitMessage(stmap))
}

func TestDeviceAggregateServer_SaveConfig(t *testing.T) {
	dir := t.TempDir()
	config := "foobar"
	given := &core.SaveConfigRequest{
		Device: "device1",
		Config: &config,
	}

	s := core.NewDeviceAggregateServer(&core.DeviceAggregateCfg{
		RootCfg: core.RootCfg{StatusRootPath: dir},
	})
	err := s.SaveConfig(context.Background(), given)
	assert.Nil(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "devices", "device1", "actual_config.cue"))
	testhelper.ExitOnErr(t, err)
	assert.Equal(t, []byte(config), got)
}

func TestDeviceAggregateServer_GitPushSyncBranch(t *testing.T) {
	testRemote := "test-remote"

	t.Run("ok", func(t *testing.T) {
		repo, dir, _ := core.SetupGitRepoWithRemote(t, testRemote)
		oldRef, _ := repo.Head()
		assert.Greater(t, len(githelper.GetStatus(t, repo)), 0)

		s := core.NewDeviceAggregateServer(&core.DeviceAggregateCfg{
			RootCfg: core.RootCfg{
				StatusRootPath: dir,
				GitRemote:      testRemote,
			},
		})
		err := s.GitPushDeviceConfig(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, len(githelper.GetStatus(t, repo)), 0)

		localRef, _ := repo.Head()
		remoteRef := githelper.GetRemoteBranch(t, repo, testRemote, "main")
		assert.NotEqual(t, localRef.Hash().String(), oldRef.Hash().String())
		assert.Equal(t, localRef.Hash().String(), remoteRef.Hash().String())
	})
}

func TestDeviceAggregateServer_Run(t *testing.T) {
	testRemote := "test-remote"
	repo, dir, _ := core.SetupGitRepoWithRemote(t, testRemote)
	config := "foobar"
	req := core.SaveConfigRequest{
		Device: "device1",
		Config: &config,
	}

	s := core.NewDeviceAggregateServer(&core.DeviceAggregateCfg{
		RootCfg: core.RootCfg{
			StatusRootPath: dir,
			GitRemote:      testRemote,
		},
	})
	core.UpdateCheckDuration = 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Run(ctx)

	buf, err := json.Marshal(req)
	testhelper.ExitOnErr(t, err)
	request := httptest.NewRequest(http.MethodPost, "/commit", bytes.NewBuffer(buf))
	response := httptest.NewRecorder()
	s.HandleFunc(response, request)
	res := response.Result()
	assert.Equal(t, 200, res.StatusCode)

	var got []byte
	assert.Eventually(t, func() bool {
		got, err = os.ReadFile(filepath.Join(dir, "devices", "device1", "actual_config.cue"))
		return err == nil
	}, time.Second, 100*time.Millisecond)
	assert.Equal(t, []byte(config), got)

	assert.Greater(t, len(githelper.GetStatus(t, repo)), 0)
	assert.Eventually(t, func() bool {
		localRef, _ := repo.Head()
		remoteRef := githelper.GetRemoteBranch(t, repo, testRemote, "main")
		return localRef.Hash().String() == remoteRef.Hash().String()
	}, time.Second, 100*time.Millisecond)

	assert.Equal(t, len(githelper.GetStatus(t, repo)), 0)
}
