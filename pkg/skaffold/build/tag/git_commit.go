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
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
	variant int
}

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
	default:
		return nil, fmt.Errorf("%s is no valid git tagger variant", taggerVariant)
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
	default:
		return "", fmt.Errorf("this should not happen, please raise an issue")
	}

	return runGit(workingDir, args...)
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
