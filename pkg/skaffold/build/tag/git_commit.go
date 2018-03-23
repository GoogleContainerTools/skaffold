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
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
)

// GitCommit tags an image by the git commit it was built at.
type GitCommit struct {
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the git commit.
func (c *GitCommit) GenerateFullyQualifiedImageName(workingDir string, opts *TagOptions) (string, error) {
	workingDir, err := findTopLevelGitDir(workingDir)
	if err != nil {
		return "", errors.Wrap(err, "invalid working dir")
	}

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

	// The file state is dirty. To generate a unique suffix, let's hash all the modified files.
	// We add a -dirty-unique-id suffix to work well with local iterations.
	h := sha256.New()
	for path, change := range status {
		if change.Worktree == git.Unmodified {
			continue
		}

		f, err := os.Open(filepath.Join(workingDir, path))
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

	return fmt.Sprintf("%s-dirty-%s", fqn, shaStr), nil
}

func findTopLevelGitDir(workingDir string) (string, error) {
	dir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", errors.Wrap(err, "invalid working dir")
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no git repository found")
		}
		dir = parent
	}
}
