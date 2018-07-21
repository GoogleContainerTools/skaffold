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
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct{}

// Labels are labels specific to the git tagger.
func (c *GitCommit) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "git-commit",
	}
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the git commit.
func (c *GitCommit) GenerateFullyQualifiedImageName(workingDir string, opts *Options) (string, error) {
	hash, err := runGit(workingDir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return fallbackOnDigest(opts, err), nil
	}

	changes, err := runGit(workingDir, "status", ".", "--porcelain")
	if err != nil {
		return "", errors.Wrap(err, "getting git status")
	}

	if len(changes) > 0 {
		return dirtyTag(hash, opts), nil
	}

	// Ignore error. It means there's no tag.
	tag, _ := runGit(workingDir, "describe", "--tags", "--exact-match")

	return commitOrTag(hash, tag, opts), nil
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

func commitOrTag(currentTag string, tag string, opts *Options) string {
	if len(tag) > 0 {
		currentTag = tag
	}

	return fmt.Sprintf("%s:%s", opts.ImageName, currentTag)
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
