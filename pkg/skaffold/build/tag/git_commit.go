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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
}

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
		return "", errors.Wrap(err, "getting git root")
	}

	revision, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return "", errors.Wrap(err, "getting current revision")
	}

	status, err := runGit(root, "status", "--porcelain")
	if err != nil {
		return "", errors.Wrap(err, "getting git status")
	}

	currentTag := revision[0:7]
	if status == "" {
		tags, err := runGit(root, "describe", "--tags", "--always")
		if err != nil {
			return "", errors.Wrap(err, "getting tags")
		}

		return commitOrTag(currentTag, lines(tags), opts), nil
	}

	return dirtyTag(root, opts, currentTag, lines(status))
}

func generateNameGoGit(workingDir string, opts *Options) (string, error) {
	repo, err := git.PlainOpenWithOptions(workingDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return "", errors.Wrap(err, "opening git repo")
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", errors.Wrap(err, "reading worktree")
	}
	root := w.Filesystem.Root()

	head, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "determining current git commit")
	}

	commitHash := head.Hash().String()
	currentTag := commitHash[0:7]

	status, err := w.Status()
	if err != nil {
		return "", errors.Wrap(err, "reading status")
	}

	if status.IsClean() {
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

	return dirtyTag(root, opts, currentTag, changes(status))
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

func commitOrTag(currentTag string, tags []string, opts *Options) string {
	if len(tags) > 0 {
		currentTag = tags[0]
	}

	return fmt.Sprintf("%s:%s", opts.ImageName, currentTag)
}

// The file state is dirty. To generate a unique suffix, let's hash all the modified files.
// We add a -dirty-unique-id suffix to work well with local iterations.
func dirtyTag(root string, opts *Options, currentTag string, lines []string) (string, error) {
	h := sha256.New()
	for _, statusLine := range lines {
		if strings.HasPrefix(statusLine, "??") {
			statusLine = statusLine[1:]
		}

		if _, err := h.Write([]byte(statusLine)); err != nil {
			return "", errors.Wrap(err, "adding deleted file to diff")
		}

		if strings.HasPrefix(statusLine, "D") {
			continue
		}

		changedPath := strings.Trim(statusLine[2:], " ")
		f, err := os.Open(filepath.Join(root, changedPath))
		if err != nil {
			return "", errors.Wrap(err, "reading diff")
		}

		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return "", errors.Wrap(err, "reading diff")
		}

		f.Close()
	}

	sha := h.Sum(nil)
	shaStr := hex.EncodeToString(sha[:])[:16]
	fqn := fmt.Sprintf("%s:%s-dirty-%s", opts.ImageName, currentTag, shaStr)
	return fqn, nil
}

func lines(text string) []string {
	var lines []string

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

// changes returns the same output as git status --porcelain.
// The order is important because we generate a sha256 out of it.
func changes(status git.Status) []string {
	var changes []string

	for path, change := range status {
		if change.Worktree != git.Unmodified {
			changes = append(changes, path)
		}
	}

	sort.Strings(changes)

	var lines []string
	for _, changedPath := range changes {
		status := status[changedPath].Worktree
		lines = append(lines, fmt.Sprintf("%c %s", status, changedPath))
	}

	return lines
}
