// code modified from https://github.com/GoogleContainerTools/kpt/blob/master/internal/gitutil/gitutil.go

// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package git

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// SyncRepo syncs the target git repository with skaffold's local cache and returns the path to the repository root directory.
var SyncRepo = syncRepo
var findGit = func() (string, error) { return exec.LookPath("git") }

// defaultRef returns the default ref as "master" if master branch exists in
// remote repository, falls back to "main" if master branch doesn't exist
func defaultRef(repo string) (string, error) {
	masterRef := "master"
	mainRef := "main"
	masterExists, err := branchExists(repo, masterRef)
	if err != nil {
		return "", err
	}
	mainExists, err := branchExists(repo, mainRef)
	if err != nil {
		return "", err
	}
	if masterExists {
		return masterRef, nil
	} else if mainExists {
		return mainRef, nil
	}
	return "", fmt.Errorf("failed to get default branch for repo %s", repo)
}

// BranchExists checks if branch is present in the input repo
func branchExists(repo, branch string) (bool, error) {
	gitProgram, err := findGit()
	if err != nil {
		return false, err
	}
	out, err := util.RunCmdOut(exec.Command(gitProgram, "ls-remote", repo, branch))
	if err != nil {
		// stdErr contains the error message for os related errors, git permission errors
		// and if repo doesn't exist
		return false, fmt.Errorf("failed to lookup %s branch for repo %s: %w", branch, repo, err)
	}
	// stdOut contains the branch information if the branch is present in remote repo
	// stdOut is empty if the repo doesn't have the input branch
	if strings.TrimSpace(string(out)) != "" {
		return true, nil
	}
	return false, nil
}

// getRepoDir returns the cache directory name for a remote repo
func getRepoDir(g latest.GitInfo) (string, error) {
	inputs := []string{g.Repo, g.Ref}
	hasher := sha256.New()
	enc := json.NewEncoder(hasher)
	if err := enc.Encode(inputs); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))[:32], nil
}

func getRepoCacheDir(opts config.SkaffoldOptions) (string, error) {
	if opts.RepoCacheDir != "" {
		return opts.RepoCacheDir, nil
	}

	// cache location unspecified, use ~/.skaffold/repos
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("retrieving home directory: %w", err)
	}
	return filepath.Join(home, constants.DefaultSkaffoldDir, "repos"), nil
}

func syncRepo(g latest.GitInfo, opts config.SkaffoldOptions) (string, error) {
	skaffoldCacheDir, err := getRepoCacheDir(opts)
	r := gitCmd{Dir: skaffoldCacheDir}
	if err != nil {
		return "", fmt.Errorf("failed to clone repo %s: %w", g.Repo, err)
	}
	if err := os.MkdirAll(skaffoldCacheDir, 0700); err != nil {
		return "", fmt.Errorf(
			"failed to clone repo %s: trouble creating cache directory: %w", g.Repo, err)
	}

	ref := g.Ref
	if ref == "" {
		ref, err = defaultRef(g.Repo)
		if err != nil {
			return "", fmt.Errorf("failed to clone repo %s: trouble getting default branch: %w", g.Repo, err)
		}
	}

	hash, err := getRepoDir(g)
	if err != nil {
		return "", fmt.Errorf("failed to clone git repo: unable to create directory name: %w", err)
	}
	repoCacheDir := filepath.Join(skaffoldCacheDir, hash)
	if _, err := os.Stat(repoCacheDir); os.IsNotExist(err) {
		if _, err := r.Run("clone", g.Repo, hash, "--branch", ref, "--depth", "1"); err != nil {
			return "", fmt.Errorf("failed to clone repo: %w", err)
		}
	} else {
		r.Dir = repoCacheDir
		// check remote is defined
		if remotes, err := r.Run("remote", "-v"); err != nil {
			return "", fmt.Errorf("failed to clone repo %s: trouble checking repository remote; run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", g.Repo, err)
		} else if len(remotes) == 0 {
			return "", fmt.Errorf("failed to clone repo %s: remote not set for existing clone", g.Repo)
		}

		// if sync property is false, then skip fetching latest from remote and resetting the branch.
		if g.Sync != nil && !*g.Sync {
			return repoCacheDir, nil
		}

		if _, err = r.Run("fetch", "origin", ref); err != nil {
			return "", fmt.Errorf("failed to clone repo %s: unable to find any matching refs %s; run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", g.Repo, ref, err)
		}

		// check if the downloaded repo has uncommitted changes.
		if changes, err := r.Run("diff", "--name-only", "--ignore-submodules", "HEAD"); err != nil {
			return "", fmt.Errorf("failed to clone repo %s: unable to check for uncommitted changes; run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", g.Repo, err)
		} else if len(changes) > 0 {
			return "", fmt.Errorf("failed to clone repo %s: there are uncommitted changes in the target directory %s; either set the repository `sync` property to false in the skaffold config, or manually commit and sync changes to remote, or revert the local changes", g.Repo, repoCacheDir)
		}

		// check if the downloaded repo has unpushed commits.
		if changes, err := r.Run("diff", "--name-only", "--ignore-submodules", fmt.Sprintf("origin/%s...", ref)); err != nil {
			return "", fmt.Errorf("failed to clone repo %s: unable to check for unpushed commits; run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", g.Repo, err)
		} else if len(changes) > 0 {
			return "", fmt.Errorf("failed to clone repo %s: there are unpushed commits in the target directory %s; either set the repository `sync` property to false in the skaffold config, or manually push commits to remote, or reset the local commits", g.Repo, repoCacheDir)
		}

		// reset the repo state
		if _, err := r.Run("reset", "--hard", fmt.Sprintf("origin/%s", ref)); err != nil {
			return "", fmt.Errorf("failed to clone repo %s: trouble resetting branch to origin/%s; run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", g.Repo, ref, err)
		}
	}
	return repoCacheDir, nil
}

// gitCmd runs git commands in a git repo.
type gitCmd struct {
	// Dir is the directory the commands are run in.
	Dir string
}

// Run runs a git command.
// Omit the 'git' part of the command.
func (g *gitCmd) Run(args ...string) ([]byte, error) {
	p, err := findGit()
	if err != nil {
		return nil, fmt.Errorf("no 'git' program on path: %w", err)
	}

	cmd := exec.Command(p, args...)
	cmd.Dir = g.Dir
	return util.RunCmdOut(cmd)
}
