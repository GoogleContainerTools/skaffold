/*
Copyright 2018 The Skaffold Authors

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

package tag

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct{}

func (c *GitCommit) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "git-commit",
	}
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the git commit.
func (c *GitCommit) GenerateFullyQualifiedImageName(workingDir string, opts *Options) (string, error) {
	if _, err := exec.LookPath("git"); err == nil {
		return generateNameGitShellOut(workingDir, opts)
	}

	logrus.Warn("git binary not found. Falling back on a go git implementation. Some features might not work.")
	return generateNameGoGit(workingDir, opts)
}

func generateNameGitShellOut(workingDir string, opts *Options) (string, error) {
	root, err := runGit(workingDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return fallbackOnDigest(opts, err), nil
	}

	commitHash, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return fallbackOnDigest(opts, err), nil
	}

	currentTag := commitHash[0:7]

	status, err := runGitLines(root, "status", "--porcelain")
	if err != nil {
		return "", errors.Wrap(err, "getting git status")
	}

	dirty, err := isDirty(root, workingDir, stripStatus(status))
	if err != nil {
		return "", errors.Wrap(err, "getting status for workingDir")
	}

	if dirty {
		return dirtyTag(currentTag, opts), nil
	}

	// Ignore error. It means there's no tag.
	tags, _ := runGitLines(root, "describe", "--tags", "--exact-match")

	return commitOrTag(currentTag, tags, opts), nil
}

func generateNameGoGit(workingDir string, opts *Options) (string, error) {
	repo, err := git.PlainOpenWithOptions(workingDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return fallbackOnDigest(opts, err), nil
	}

	w, err := repo.Worktree()
	if err != nil {
		return fallbackOnDigest(opts, err), nil
	}
	root := w.Filesystem.Root()

	head, err := repo.Head()
	if err != nil {
		return fallbackOnDigest(opts, err), nil
	}

	commitHash := head.Hash().String()
	currentTag := commitHash[0:7]

	status, err := w.Status()
	if err != nil {
		return "", errors.Wrap(err, "reading status")
	}

	dirty, err := isDirty(root, workingDir, changedPaths(status))
	if err != nil {
		return "", errors.Wrap(err, "getting status for workingDir")
	}

	if dirty {
		return dirtyTag(currentTag, opts), nil
	}

	tagrefs, err := repo.Tags()
	if err != nil {
		return "", errors.Wrap(err, "determining git tag")
	}

	var tags []string
	if err = tagrefs.ForEach(func(t *plumbing.Reference) error {
		if t.Hash() == head.Hash() {
			tags = append(tags, t.Name().Short())
		}
		return nil
	}); err != nil {
		return "", errors.Wrap(err, "determining git tag")
	}

	return commitOrTag(currentTag, tags, opts), nil
}

func runGit(workingDir string, arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = workingDir

	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func runGitLines(workingDir string, arg ...string) ([]string, error) {
	out, err := runGit(workingDir, arg...)
	if err != nil {
		return nil, err
	}

	var lines []string

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines, nil
}

func commitOrTag(currentTag string, tags []string, opts *Options) string {
	if len(tags) > 0 {
		currentTag = tags[0]
	}

	return fmt.Sprintf("%s:%s", opts.ImageName, currentTag)
}

func stripStatus(lines []string) []string {
	var paths []string

	for _, line := range lines {
		path := strings.Fields(line)[1]
		paths = append(paths, path)
	}

	return paths
}

func isDirty(root, workingDir string, changes []string) (bool, error) {
	root, err := normalizePath(root)
	if err != nil {
		return false, errors.Wrap(err, "normalizing path")
	}

	absWorkingDir, err := normalizePath(workingDir)
	if err != nil {
		return false, errors.Wrap(err, "normalizing path")
	}

	for _, change := range changes {
		if strings.HasPrefix(filepath.Join(root, change), absWorkingDir) {
			return true, nil
		}
	}

	return false, nil
}

func shortDigest(opts *Options) string {
	return strings.TrimPrefix(opts.Digest, "sha256:")[0:7]
}

func dirtyTag(currentTag string, opts *Options) string {
	return fmt.Sprintf("%s:%s-dirty-%s", opts.ImageName, currentTag, shortDigest(opts))
}

func fallbackOnDigest(opts *Options, err error) string {
	logrus.Warnln("Using digest instead of git commit:", err)

	return fmt.Sprintf("%s:dirty-%s", opts.ImageName, shortDigest(opts))
}

func changedPaths(status git.Status) []string {
	var paths []string

	for path, change := range status {
		if change.Worktree != git.Unmodified {
			paths = append(paths, path)
		}
	}

	return paths
}

func normalizePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
