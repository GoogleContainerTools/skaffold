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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	tags = iota
	commitSha
	abbrevCommitSha
	treeSha
	abbrevTreeSha
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
	variant int
}

// NewGitCommit creates a new git commit tagger. It fails if the tagger variant is invalid.
func NewGitCommit(taggerVariant string) (*GitCommit, error) {
	var variant int
	switch strings.ToLower(taggerVariant) {
	case "", "tags":
		// default to "tags" when unset
		variant = tags
	case "commitsha":
		variant = commitSha
	case "abbrevcommitsha":
		variant = abbrevCommitSha
	case "treesha":
		variant = treeSha
	case "abbrevtreesha":
		variant = abbrevTreeSha
	default:
		return nil, fmt.Errorf("%s is not a valid git tagger variant", taggerVariant)
	}

	return &GitCommit{variant: variant}, nil
}

// Labels are labels specific to the git tagger.
func (c *GitCommit) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "git-commit",
	}
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the git commit.
func (c *GitCommit) GenerateFullyQualifiedImageName(workingDir string, imageName string) (string, error) {
	ref, err := c.makeGitTag(workingDir)
	if err != nil {
		logrus.Warnln("Unable to find git commit:", err)
		return fmt.Sprintf("%s:dirty", imageName), nil
	}

	changes, err := runGit(workingDir, "status", ".", "--porcelain")
	if err != nil {
		return "", errors.Wrap(err, "getting git status")
	}

	if len(changes) > 0 {
		return fmt.Sprintf("%s:%s-dirty", imageName, ref), nil
	}

	return fmt.Sprintf("%s:%s", imageName, ref), nil
}

func (c *GitCommit) makeGitTag(workingDir string) (string, error) {
	args := make([]string, 0, 4)
	switch c.variant {
	case tags:
		args = append(args, "describe", "--tags", "--always")
	case commitSha, abbrevCommitSha:
		args = append(args, "rev-list", "-1", "HEAD")
		if c.variant == abbrevCommitSha {
			args = append(args, "--abbrev-commit")
		}
	case treeSha, abbrevTreeSha:
		gitPath, err := getGitPathToWorkdir(workingDir)
		if err != nil {
			return "", err
		}
		args = append(args, "rev-parse")
		if c.variant == abbrevTreeSha {
			args = append(args, "--short")
		}
		// revision must come after the --short flag
		args = append(args, "HEAD:"+gitPath+"/")
	default:
		return "", errors.New("invalid git tag variant: defaulting to 'dirty'")
	}

	return runGit(workingDir, args...)
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
