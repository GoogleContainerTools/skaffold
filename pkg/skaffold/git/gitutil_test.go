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
	"errors"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
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
				f = testutil.CmdRunOut("git ls-remote https://github.com/foo.git master", "8be3f718c015a5fe190bebf356079a25afe0ca57  refs/heads/master")
			} else {
				f = testutil.CmdRunOut("git ls-remote https://github.com/foo.git master", "")
			}
			if test.mainExists {
				f = f.AndRunOut("git ls-remote https://github.com/foo.git main", "8be3f718c015a5fe190bebf356079a25afe0ca58  refs/heads/main")
			} else {
				f = f.AndRunOut("git ls-remote https://github.com/foo.git main", "")
			}
			t.Override(&findGit, func() (string, error) { return "git", nil })
			t.Override(&util.DefaultExecCommand, f)
			ref, err := defaultRef("https://github.com/foo.git")
			t.CheckErrorAndDeepEqual(test.err != nil, err, test.expected, ref)
		})
	}
}

func TestSyncRepo(t *testing.T) {
	tests := []struct {
		description string
		g           latest.GitInfo
		cmds        []cmdResponse
		existing    bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "first time repo clone succeeds",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			cmds: []cmdResponse{
				{cmd: "git clone http://github.com/foo.git iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl --branch master --depth 1"},
			},
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "first time repo clone fails",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			cmds: []cmdResponse{
				{cmd: "git clone http://github.com/foo.git iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl --branch master --depth 1", err: errors.New("error")},
			},
			shouldErr: true,
		},
		{
			description: "existing repo update succeeds",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git diff --name-only --ignore-submodules HEAD"},
				{cmd: "git diff --name-only --ignore-submodules origin/master..."},
				{cmd: "git reset --hard origin/master"},
			},
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "existing repo update fails on remote check",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", err: errors.New("error")},
			},
			shouldErr: true,
		},
		{
			description: "existing dirty repo with sync off succeeds",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master", Sync: util.BoolPtr(false)},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
			},
			expected: "iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl",
		},
		{
			description: "existing repo with uncommitted changes and sync on fails",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master", Sync: util.BoolPtr(true)},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git diff --name-only --ignore-submodules HEAD", out: "pkg/foo\npkg/bar"},
			},
			shouldErr: true,
		},
		{
			description: "existing repo with unpushed commits and sync on fails",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master", Sync: util.BoolPtr(true)},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git diff --name-only --ignore-submodules HEAD"},
				{cmd: "git diff --name-only --ignore-submodules origin/master...", out: "pkg/foo\npkg/bar"},
				{cmd: "git reset --hard origin/master"},
			},
			shouldErr: true,
		},
		{
			description: "existing repo update fails on fetch",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master", err: errors.New("error")},
			},
			shouldErr: true,
		},
		{
			description: "existing repo update fails on diff remote",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git diff --name-only --ignore-submodules HEAD"},
				{cmd: "git diff --name-only --ignore-submodules origin/master...", err: errors.New("error")},
			},
			shouldErr: true,
		},
		{
			description: "existing repo update fails on diff working dir",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git diff --name-only --ignore-submodules HEAD", err: errors.New("error")},
			},
			shouldErr: true,
		},
		{
			description: "existing repo update fails on reset",
			g:           latest.GitInfo{Repo: "http://github.com/foo.git", Path: "bar/skaffold.yaml", Ref: "master"},
			existing:    true,
			cmds: []cmdResponse{
				{cmd: "git remote -v", out: "origin git@github.com/foo.git"},
				{cmd: "git fetch origin master"},
				{cmd: "git diff --name-only --ignore-submodules HEAD"},
				{cmd: "git diff --name-only --ignore-submodules origin/master..."},
				{cmd: "git reset --hard origin/master", err: errors.New("error")},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			td := t.NewTempDir()
			if test.existing {
				td.Touch("iSEL5rQfK5EJ2yLhnW8tUgcVOvDC8Wjl/.git/")
			}
			opts := config.SkaffoldOptions{RepoCacheDir: td.Root()}
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
			path, err := syncRepo(test.g, opts)
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
