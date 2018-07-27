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

package watch

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const masterRef = "refs/heads/master"

type GitFileWatcher struct {
	url          string
	pollInterval time.Duration

	// TODO(r2d4): watch pulls and tags
}

func NewGitFileWatcher(url string, pollInterval time.Duration) FileWatcher {
	logrus.Infof("Using git file watcher: %s", url)
	return &GitFileWatcher{
		url:          url,
		pollInterval: pollInterval,
	}
}

func (g *GitFileWatcher) Run(ctx context.Context, callback FileChangedFn) error {
	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()
	lastCommit, err := getHeadRef()
	if err != nil {
		return errors.Wrap(err, "getting initial head ref")
	}
	for {
		select {
		case <-ticker.C:
			currentCommit, err := getCurrentCommit(g.url)
			if err != nil {
				return errors.Wrap(err, "getting current commit")
			}
			logrus.Debugf("Current commit %s, Last commit built: %s", currentCommit, lastCommit)
			if lastCommit == currentCommit {
				continue
			}

			fetchCmd := exec.Command("git", "fetch", "origin", "master")
			if err := util.RunCmd(fetchCmd); err != nil {
				return errors.Wrap(err, "fetching head")
			}

			diff, err := computeGitDiff(lastCommit, currentCommit)
			if err != nil {
				return errors.Wrap(err, "computing diff")
			}

			checkoutCmd := exec.Command("git", "checkout", "-f", currentCommit)
			if err := util.RunCmd(checkoutCmd); err != nil {
				return errors.Wrap(err, "checking out latest commit")
			}

			lastCommit = currentCommit

			if len(diff) > 0 {
				if err := callback(diff); err != nil {
					return errors.Wrap(err, "git watcher callback")
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func computeGitDiff(srcRef, targetRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", srcRef, targetRef)
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "diffing revisions")
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

func getCurrentCommit(url string) (string, error) {
	cmd := exec.Command("git", "ls-remote", url, masterRef)
	lsRemoteOutput, err := util.RunCmdOut(cmd)
	if err != nil {
		return "", errors.Wrap(err, "getting latest commit reference")
	}
	commitAndRef := strings.Split(string(lsRemoteOutput), "\t")
	return strings.TrimSpace(commitAndRef[0]), nil
}

func getHeadRef() (string, error) {
	headCmd := exec.Command("git", "rev-parse", "master")
	head, err := util.RunCmdOut(headCmd)
	if err != nil {
		return "", errors.Wrap(err, "getting initial head ref")
	}
	return strings.TrimSpace(string(head)), nil
}
