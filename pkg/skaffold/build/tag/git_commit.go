/*
Copyright 2019 The Skaffold Authors

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
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
	prefix   string
	runGitFn func(string) (string, error)
}

var variants = map[string]func(string) (string, error){
	"":                gitTags,
	"tags":            gitTags,
	"commitsha":       gitCommitsha,
	"abbrevcommitsha": gitAbbrevcommitsha,
	"treesha":         gitTreesha,
	"abbrevtreesha":   gitAbbrevtreesha,
}

// NewGitCommit creates a new git commit tagger. It fails if the tagger variant is invalid.
func NewGitCommit(prefix, variant string) (*GitCommit, error) {
	runGitFn, found := variants[strings.ToLower(variant)]
	if !found {
		return nil, fmt.Errorf("%q is not a valid git tagger variant", variant)
	}

	return &GitCommit{
		prefix:   prefix,
		runGitFn: runGitFn,
	}, nil
}

// GenerateTag generates a tag from the git commit.
func (t *GitCommit) GenerateTag(workingDir, _ string) (string, error) {
	ref, err := t.runGitFn(workingDir)
	if err != nil {
		return "", fmt.Errorf("unable to find git commit: %w", err)
	}

	changes, err := runGit(workingDir, "status", ".", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("getting git status: %w", err)
	}

	if len(changes) > 0 {
		return fmt.Sprintf("%s%s-dirty", t.prefix, ref), nil
	}

	return t.prefix + sanitizeTag(ref), nil
}

// sanitizeTag takes a git tag and converts it to a docker tag by removing
// all the characters that are not allowed by docker.
func sanitizeTag(tag string) string {
	// Replace unsupported characters with `_`
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9-._]`).ReplaceAllString(tag, `_`)

	// Remove leading `-`s and `.`s
	prefixSuffix := regexp.MustCompile(`([-.]*)(.*)`).FindStringSubmatch(sanitized)
	sanitized = strings.Repeat("_", len(prefixSuffix[1])) + prefixSuffix[2]

	// Truncate to 128 characters
	if len(sanitized) > 128 {
		return sanitized[0:128]
	}

	if tag != sanitized {
		logrus.Warnf("Using %q instead of %q as an image tag", sanitized, tag)
	}

	return sanitized
}

func gitTags(workingDir string) (string, error) {
	return runGit(workingDir, "describe", "--tags", "--always")
}

func gitCommitsha(workingDir string) (string, error) {
	return runGit(workingDir, "rev-list", "-1", "HEAD")
}

func gitAbbrevcommitsha(workingDir string) (string, error) {
	return runGit(workingDir, "rev-list", "-1", "HEAD", "--abbrev-commit")
}

func gitTreesha(workingDir string) (string, error) {
	gitPath, err := getGitPathToWorkdir(workingDir)
	if err != nil {
		return "", err
	}

	return runGit(workingDir, "rev-parse", "HEAD:"+gitPath+"/")
}

func gitAbbrevtreesha(workingDir string) (string, error) {
	gitPath, err := getGitPathToWorkdir(workingDir)
	if err != nil {
		return "", err
	}

	return runGit(workingDir, "rev-parse", "--short", "HEAD:"+gitPath+"/")
}

func getGitPathToWorkdir(workingDir string) (string, error) {
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", err
	}

	// git reports the gitdir with resolved symlinks, so we need to do this too in order for filepath.Rel to work
	absWorkingDir, err = filepath.EvalSymlinks(absWorkingDir)
	if err != nil {
		return "", err
	}

	gitRoot, err := runGit(workingDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return filepath.Rel(gitRoot, absWorkingDir)
}

func runGit(workingDir string, arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = workingDir

	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(out)), nil
}
