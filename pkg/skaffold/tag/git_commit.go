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
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
	prefix        string
	runGitFn      func(context.Context, string) (string, error)
	ignoreChanges bool
}

var variants = map[string]func(context.Context, string) (string, error){
	"":                gitTags,
	"tags":            gitTags,
	"commitsha":       gitCommitsha,
	"abbrevcommitsha": gitAbbrevcommitsha,
	"treesha":         gitTreesha,
	"abbrevtreesha":   gitAbbrevtreesha,
	"branches":        gitBranches,
}

// NewGitCommit creates a new git commit tagger. It fails if the tagger variant is invalid.
func NewGitCommit(prefix, variant string, ignoreChanges bool) (*GitCommit, error) {
	runGitFn, found := variants[strings.ToLower(variant)]
	if !found {
		return nil, fmt.Errorf("%q is not a valid git tagger variant", variant)
	}

	return &GitCommit{
		prefix:        prefix,
		runGitFn:      runGitFn,
		ignoreChanges: ignoreChanges,
	}, nil
}

// GenerateTag generates a tag from the git commit.
func (t *GitCommit) GenerateTag(ctx context.Context, image latest.Artifact) (string, error) {
	ref, err := t.runGitFn(ctx, image.Workspace)
	if err != nil {
		return "", fmt.Errorf("unable to find git commit: %w", err)
	}

	ref = sanitizeTag(ref)

	if !t.ignoreChanges {
		changes, err := runGit(ctx, image.Workspace, "status", ".", "--porcelain")
		if err != nil {
			return "", fmt.Errorf("getting git status: %w", err)
		}

		if len(changes) > 0 {
			return fmt.Sprintf("%s%s-dirty", t.prefix, ref), nil
		}
	}

	return t.prefix + ref, nil
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
		log.Entry(context.TODO()).Warnf("Using %q instead of %q as an image tag", sanitized, tag)
	}

	return sanitized
}

func gitTags(ctx context.Context, workingDir string) (string, error) {
	return runGit(ctx, workingDir, "describe", "--tags", "--always")
}

func gitCommitsha(ctx context.Context, workingDir string) (string, error) {
	return runGit(ctx, workingDir, "rev-list", "-1", "HEAD")
}

func gitAbbrevcommitsha(ctx context.Context, workingDir string) (string, error) {
	return runGit(ctx, workingDir, "rev-list", "-1", "HEAD", "--abbrev-commit")
}

func gitTreesha(ctx context.Context, workingDir string) (string, error) {
	gitPath, err := getGitPathToWorkdir(ctx, workingDir)
	if err != nil {
		return "", err
	}

	return runGit(ctx, workingDir, "rev-parse", "HEAD:"+gitPath+"/")
}

func gitAbbrevtreesha(ctx context.Context, workingDir string) (string, error) {
	gitPath, err := getGitPathToWorkdir(ctx, workingDir)
	if err != nil {
		return "", err
	}

	return runGit(ctx, workingDir, "rev-parse", "--short", "HEAD:"+gitPath+"/")
}

func gitBranches(ctx context.Context, workingDir string) (string, error) {
	gitBranch, err := runGit(ctx, workingDir, "branch", "--show-current")
	if err != nil {
		return gitAbbrevcommitsha(ctx, workingDir)
	}

	return gitBranch, nil
}

func getGitPathToWorkdir(ctx context.Context, workingDir string) (string, error) {
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", err
	}

	// git reports the gitdir with resolved symlinks, so we need to do this too in order for filepath.Rel to work
	absWorkingDir, err = filepath.EvalSymlinks(absWorkingDir)
	if err != nil {
		return "", err
	}

	gitRoot, err := runGit(ctx, workingDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return filepath.Rel(gitRoot, absWorkingDir)
}

func runGit(ctx context.Context, workingDir string, arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = workingDir

	out, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(out)), nil
}
