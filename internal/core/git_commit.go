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

package core

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/nttcom/kuesta/internal/gitrepo"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/internal/logger"
	error2 "github.com/nttcom/kuesta/internal/util"
	"github.com/nttcom/kuesta/internal/validator"
	"go.uber.org/multierr"
)

type GitCommitCfg struct {
	RootCfg
}

// Validate validates exposed fields according to the `validate` tag.
func (c *GitCommitCfg) Validate() error {
	return validator.Validate(c)
}

// Mask returns the copy whose sensitive data are masked.
func (c *GitCommitCfg) Mask() *GitCommitCfg {
	cc := *c
	cc.RootCfg = *c.RootCfg.Mask()
	return &cc
}

// RunGitCommit runs the main process of the `git commit` command.
func RunGitCommit(ctx context.Context, cfg *GitCommitCfg) error {
	l := logger.FromContext(ctx)
	out := WriterFromContext(ctx)
	l.Debugw("git commit called", "config", cfg.Mask())

	git, err := gogit.NewGit(cfg.ConfigGitOptions())
	if err != nil {
		return fmt.Errorf("setup git: %w", err)
	}

	w, err := git.Checkout()
	if err != nil {
		return fmt.Errorf("git checkout: %w", err)
	}
	stmap, err := w.Status()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(stmap) == 0 {
		fmt.Fprintf(out, "Skipped: There are no update.")
		return nil
	}
	if err := CheckGitIsStagedOrUnmodified(stmap); err != nil {
		return fmt.Errorf("check files are either staged or unmodified: %w", err)
	}

	t := time.Now()
	branchName := cfg.GitTrunk
	if !cfg.PushToMain {
		branchName = fmt.Sprintf("REV-%d", t.Unix())
		if _, err = git.Checkout(gogit.CheckoutOptsTo(branchName), gogit.CheckoutOptsCreateNew(), gogit.CheckoutOptsSoftReset()); err != nil {
			return fmt.Errorf("create new branch: %w", err)
		}
	}

	commitMsg := MakeCommitMessage(stmap)
	if _, err := git.Commit(commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	if err := git.Push(gogit.PushOptBranch(branchName)); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	if !cfg.PushToMain {
		c, err := gitrepo.NewGitRepoClient(cfg.ConfigRepoUrl, cfg.GitToken)
		if err != nil {
			return fmt.Errorf("create pull-request from branch=%s: %w", branchName, err)
		}
		payload := gitrepo.GitPullRequestPayload{
			HeadRef: branchName,
			BaseRef: cfg.GitTrunk,
			Title:   "[kuesta] Automated PR",
			Body:    commitMsg,
		}
		if _, err := c.CreatePullRequest(ctx, payload); err != nil {
			return fmt.Errorf("create pull-request from branch=%s: %w", branchName, err)
		}
	}

	return nil
}

// MakeCommitMessage returns the commit message that shows the summary of service and device updates.
func MakeCommitMessage(stmap extgogit.Status) string {
	var servicesAdded []string
	var servicesModified []string
	var servicesDeleted []string
	var devicesAdded []string
	var devicesModified []string
	var devicesDeleted []string

	for path, st := range stmap {
		dir, file := filepath.Split(path)
		dirElem := strings.Split(dir, string(filepath.Separator))
		if dirElem[0] == "services" && file == "input.cue" {
			serviceName := strings.TrimRight(dir, string(filepath.Separator))
			switch st.Staging {
			case extgogit.Added:
				servicesAdded = append(servicesAdded, serviceName)
			case extgogit.Modified:
				servicesModified = append(servicesModified, serviceName)
			case extgogit.Deleted:
				servicesDeleted = append(servicesDeleted, serviceName)
			default:
				// noop
			}
		}
		if dirElem[0] == "devices" && file == "config.cue" {
			deviceName := dirElem[1]
			switch st.Staging {
			case extgogit.Added:
				devicesAdded = append(devicesAdded, deviceName)
			case extgogit.Modified:
				devicesModified = append(devicesModified, deviceName)
			case extgogit.Deleted:
				devicesDeleted = append(devicesDeleted, deviceName)
			default:
				// noop
			}
		}
	}
	for _, v := range [][]string{servicesAdded, servicesModified, servicesDeleted, devicesAdded, devicesModified, devicesDeleted} {
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}

	var services []string
	services = append(services, servicesAdded...)
	services = append(services, servicesDeleted...)
	services = append(services, servicesModified...)

	title := fmt.Sprintf("Updated: %s", strings.Join(services, " "))
	var bodylines []string
	bodylines = append(bodylines, "", "Services:")
	for _, s := range servicesAdded {
		bodylines = append(bodylines, fmt.Sprintf("\tadded:     %s", s))
	}
	for _, s := range servicesDeleted {
		bodylines = append(bodylines, fmt.Sprintf("\tdeleted:   %s", s))
	}
	for _, s := range servicesModified {
		bodylines = append(bodylines, fmt.Sprintf("\tmodified:  %s", s))
	}

	bodylines = append(bodylines, "", "Devices:")
	for _, d := range devicesAdded {
		bodylines = append(bodylines, fmt.Sprintf("\tadded:     %s", d))
	}
	for _, d := range devicesDeleted {
		bodylines = append(bodylines, fmt.Sprintf("\tdeleted:   %s", d))
	}
	for _, d := range devicesModified {
		bodylines = append(bodylines, fmt.Sprintf("\tmodified:  %s", d))
	}

	return title + "\n" + strings.Join(bodylines, "\n")
}

// CheckGitIsStagedOrUnmodified checks all tracked files are modified and staged, or unmodified.
func CheckGitIsStagedOrUnmodified(stmap extgogit.Status) error {
	var err error
	for path, st := range stmap {
		err = multierr.Append(err, CheckGitFileIsStagedOrUnmodified(path, *st))
	}
	if err != nil {
		return error2.JoinErr("check git status:", err)
	}
	return nil
}

// CheckGitFileIsStagedOrUnmodified checks the given file status is modified and staged, or unmodified.
func CheckGitFileIsStagedOrUnmodified(path string, st extgogit.FileStatus) error {
	if st.Worktree == extgogit.UpdatedButUnmerged {
		return fmt.Errorf("changes conflicted: you have to solve it in advance: %s", path)
	}
	if gogit.IsBothWorktreeAndStagingTrackedAndChanged(st) {
		return fmt.Errorf("both worktree and staging are modified: only changes in worktree or staging is allowed: %s", path)
	}
	return nil
}
