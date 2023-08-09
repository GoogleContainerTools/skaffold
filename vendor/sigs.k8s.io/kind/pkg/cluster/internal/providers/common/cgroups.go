/*
Copyright 2021 The Kubernetes Authors.

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

package common

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"sync"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

var nodeReachedCgroupsReadyRegexp *regexp.Regexp
var nodeReachedCgroupsReadyRegexpCompileOnce sync.Once

// NodeReachedCgroupsReadyRegexp returns a regexp for use with WaitUntilLogRegexpMatches
//
// This is used to avoid "ERROR: this script needs /sys/fs/cgroup/cgroup.procs to be empty (for writing the top-level cgroup.subtree_control)"
// See https://github.com/kubernetes-sigs/kind/issues/2409
//
// This pattern matches either "detected cgroupv1" from the kind node image's entrypoint logs
// or "Multi-User System" target if is using cgroups v2,
// so that `docker exec` can be executed safely without breaking cgroup v2 hierarchy.
func NodeReachedCgroupsReadyRegexp() *regexp.Regexp {
	nodeReachedCgroupsReadyRegexpCompileOnce.Do(func() {
		// This is an approximation, see: https://github.com/kubernetes-sigs/kind/pull/2421
		nodeReachedCgroupsReadyRegexp = regexp.MustCompile("Reached target .*Multi-User System.*|detected cgroup v1")
	})
	return nodeReachedCgroupsReadyRegexp
}

// WaitUntilLogRegexpMatches waits until logCmd output produces a line matching re.
// It will use logCtx to determine if the logCmd deadline was exceeded for producing
// the most useful error message in failure cases, logCtx should be the context
// supplied to create logCmd with CommandContext
func WaitUntilLogRegexpMatches(logCtx context.Context, logCmd exec.Cmd, re *regexp.Regexp) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	logCmd.SetStdout(pw)
	logCmd.SetStderr(pw)

	defer pr.Close()
	cmdErrC := make(chan error, 1)
	go func() {
		defer pw.Close()
		cmdErrC <- logCmd.Run()
	}()

	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := sc.Text()
		if re.MatchString(line) {
			return nil
		}
	}

	// when we timeout the process will have been killed due to the timeout, which is not interesting
	// in other cases if the command errored this may be a useful error
	if ctxErr := logCtx.Err(); ctxErr != context.DeadlineExceeded {
		if cmdErr := <-cmdErrC; cmdErr != nil {
			return errors.Wrap(cmdErr, "failed to read logs")
		}
	}
	// otherwise generic error
	return errors.Errorf("could not find a log line that matches %q", re.String())
}
