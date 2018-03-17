/*
Copyright 2018 Google LLC

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the git commit.
func (c *GitCommit) GenerateFullyQualifiedImageName(workingDir string, opts *TagOptions) (string, error) {
	// If the repository state is dirty, we add a -dirty-unique-id suffix to work well with local iterations
	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		return "", errors.Wrap(err, "opening git repo")
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", errors.Wrap(err, "reading worktree")
	}

	status, err := w.Status()
	if err != nil {
		return "", errors.Wrap(err, "reading status")
	}

	head, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "determining current git commit")
	}

	shortCommit := head.Hash().String()[0:7]

	fqn := fmt.Sprintf("%s:%s", opts.ImageName, shortCommit)
	if status.IsClean() {
		return fqn, nil
	}

	// The file state is dirty. To generate a unique suffix, let's hash the "git diff" output.
	// It should be roughly content-addressable.
	uniqueCmd := exec.Command("git", "diff")
	uniqueCmd.Dir = workingDir
	stdout, _, err := util.RunCommand(uniqueCmd, nil)
	if err != nil {
		return "", errors.Wrap(err, "determining git diff")
	}

	sha := sha256.Sum256(stdout)
	shaStr := hex.EncodeToString(sha[:])[:16]

	return fmt.Sprintf("%s-dirty-%s", fqn, shaStr), nil
}
