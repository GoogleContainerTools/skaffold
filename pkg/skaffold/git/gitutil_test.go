/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package git

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDefaultRef(t *testing.T) {
	tests := []struct {
		description  string
		masterExists bool
		mainExists   bool
		expected     string
		err          error
	}{
		{
			description:  "master branch exists",
			masterExists: true,
			mainExists:   true,
			expected:     "master",
		},
		{
			description: "master branch does not exist; main branch exists",
			mainExists:  true,
			expected:    "main",
		},
		{
			description: "master and main don't exist",
			err:         errors.New("failed to get default branch for repo http://github.com/foo.git"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var f *testutil.FakeCmd
			if test.masterExists {
				f = testutil.CmdRunOut("git ls-remote --heads https://github.com/foo.git master", "8be3f718c015a5fe190bebf356079a25afe0ca57  refs/heads/master")
			} else {
				f = testutil.CmdRunOut("git ls-remote --heads https://github.com/foo.git master", "")
			}
			if test.mainExists {
				f = f.AndRunOut("git ls-remote --heads https://github.com/foo.git main", "8be3f718c015a5fe190bebf356079a25afe0ca58  refs/heads/main")
			} else {
				f = f.AndRunOut("git ls-remote --heads https://github.com/foo.git main", "")
			}
			t.Override(&findGit, func() (string, error) { return "git", nil })
			t.Override(&util.DefaultExecCommand, f)
			ref, err := defaultRef(context.Background(), "https://github.com/foo.git")
			t.CheckErrorAndDeepEqual(test.err != nil, err, test.expected, ref)
		})
	}
}

func TestSyncRepo(t *testing.T) {
	tests := []struct {
		description string
		g           latestV2.GitInfo
		cmds        []cmdResponse
		syncFlag    string
		existing    bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "first time repo clone succeeds",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			cmds: []cmdResponse{
				{cmd: "git clone http://github.com/foo.git ./iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl --branch master --depth 1"},
			},
			syncFlag: "always",
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "first time repo clone fails",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			cmds: []cmdResponse{
				{cmd: "git clone http://github.com/foo.git ./iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl --branch master --depth 1", err: errors.New("error")},
			},
			syncFlag:  "always",
			shouldErr: true,
		},
		{
			description: "first time repo clone with sync off via flag fails",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			syncFlag:    "never",
			shouldErr:   true,
		},
		{
			description: "existing repo update succeeds",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git reset --hard origin/master"},
			},
			syncFlag: "always",
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "existing repo update fails on remote check",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", err: errors.New("error")},
			},
			syncFlag:  "always",
			shouldErr: true,
		},
		{
			description: "existing repo with no remotes fails",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v"},
			},
			syncFlag:  "always",
			shouldErr: true,
		},
		{
			description: "existing dirty repo with sync off succeeds",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master", Sync: util.BoolPtr(false)},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
			},
			syncFlag: "always",
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "existing dirty repo with sync off via flag succeeds",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
			},
			syncFlag: "missing",
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "existing repo with unpushed commits and sync on resets",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master", Sync: util.BoolPtr(true)},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git reset --hard origin/master"},
			},
			syncFlag: "always",
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "existing repo update fails on fetch",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master", err: errors.New("error")},
			},
			syncFlag:  "always",
			shouldErr: true,
		},
		{
			description: "existing repo update fails on reset",
			g:           latestV2.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git reset --hard origin/master", err: errors.New("error")},
			},
			syncFlag:  "always",
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			td := t.NewTempDir()
			if test.existing {
				td.Touch("iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl/.git/")
			}
			syncRemote := &config.SyncRemoteCacheOption{}
			_ = syncRemote.Set(test.syncFlag)
			opts := config.SkaffoldOptions{RepoCacheDir: td.Root(), SyncRemoteCache: *syncRemote}
			var f *testutil.FakeCmd
			for _, v := range test.cmds {
				if f == nil {
					f = testutil.CmdRunOutErr(v.cmd, v.out, v.err)
				} else {
					f = f.AndRunOutErr(v.cmd, v.out, v.err)
				}
			}
			t.Override(&findGit, func() (string, error) { return "git", nil })
			t.Override(&util.DefaultExecCommand, f)
			path, err := syncRepo(context.Background(), test.g, opts)
			var expected string
			if !test.shouldErr {
				expected = filepath.Join(td.Root(), test.expected)
			}
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expected, path)
		})
	}
}

type cmdResponse struct {
	cmd string
	out string
	err error
}
