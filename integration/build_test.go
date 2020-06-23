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

package integration

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"4d63.com/tz"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const imageName = "gcr.io/k8s-skaffold/simple-build:"

func TestBuild(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		dir         string
		args        []string
		expectImage string
		setup       func(t *testing.T, workdir string)
	}{
		{
			description: "docker build",
			dir:         "testdata/build",
		},
		{
			description: "git tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "gitCommit"},
			setup:       setupGitRepo,
			expectImage: imageName + "v1",
		},
		{
			description: "sha256 tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "sha256"},
			expectImage: imageName + "latest",
		},
		{
			description: "dateTime tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "dateTime"},
			// around midnight this test might fail, if the tests above run slowly
			expectImage: imageName + nowInChicago(),
		},
		{
			description: "envTemplate tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "envTemplate"},
			expectImage: imageName + "tag",
		},
		{
			description: "custom",
			dir:         "examples/custom",
		},
		{
			description: "--tag arg",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "args", "-t", "foo"},
			expectImage: imageName + "foo",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t, test.dir)
			}

			// Run without artifact caching
			skaffold.Build(append(test.args, "--cache-artifacts=false")...).InDir(test.dir).RunOrFail(t)

			// Run with artifact caching
			skaffold.Build(append(test.args, "--cache-artifacts=true")...).InDir(test.dir).RunOrFail(t)

			// Run a second time with artifact caching
			out := skaffold.Build(append(test.args, "--cache-artifacts=true")...).InDir(test.dir).RunOrFailOutput(t)
			if strings.Contains(string(out), "Not found. Building") {
				t.Errorf("images were expected to be found in cache: %s", out)
			}
			checkImageExists(t, test.expectImage)
		})
	}
}

// TestExpectedBuildFailures verifies that `skaffold build` fails in expected ways
func TestExpectedBuildFailures(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		dir         string
		args        []string
		expected    string
	}{
		{
			description: "jib is too old",
			dir:         "testdata/jib",
			args:        []string{"-p", "old-jib"},
			expected:    "Could not find goal '_skaffold-fail-if-jib-out-of-date' in plugin com.google.cloud.tools:jib-maven-plugin:1.3.0",
			// test string will need to be updated for the jib.requiredVersion error text when moving to Jib > 1.4.0
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if out, err := skaffold.Build(test.args...).InDir(test.dir).RunWithCombinedOutput(t); err == nil {
				t.Fatal("expected build to fail")
			} else if !strings.Contains(string(out), test.expected) {
				logrus.Info("build output: ", string(out))
				t.Fatalf("build failed but for wrong reason")
			}
		})
	}
}

// checkImageExists asserts that the given image is present
func checkImageExists(t *testing.T, image string) {
	t.Helper()

	if image == "" {
		return
	}

	cfg, err := kubectx.CurrentConfig()
	failNowIfError(t, err)

	// TODO: use the proper RunContext
	client, err := docker.NewAPIClient(&runcontext.RunContext{
		KubeContext: cfg.CurrentContext,
	})
	failNowIfError(t, err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	if !client.ImageExists(ctx, image) {
		t.Errorf("expected image '%s' not present", image)
	}
}

// setupGitRepo sets up a clean repo with tag v1
func setupGitRepo(t *testing.T, dir string) {
	t.Cleanup(func() { os.RemoveAll(dir + "/.git") })

	gitArgs := [][]string{
		{"init"},
		{"config", "user.email", "john@doe.org"},
		{"config", "user.name", "John Doe"},
		{"add", "."},
		{"commit", "-m", "Initial commit"},
		{"tag", "v1"},
	}

	for _, args := range gitArgs {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if buf, err := util.RunCmdOut(cmd); err != nil {
			t.Logf(string(buf))
			t.Fatal(err)
		}
	}
}

// nowInChicago returns the dateTime string as generated by the dateTime tagger
func nowInChicago() string {
	loc, _ := tz.LoadLocation("America/Chicago")
	return time.Now().In(loc).Format("2006-01-02")
}

type Fataler interface {
	Fatal(args ...interface{})
	Helper()
}

func failNowIfError(t Fataler, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
