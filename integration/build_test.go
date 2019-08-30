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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"4d63.com/tz"
	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const imageName = "simple-build:"

func TestBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	tests := []struct {
		description string
		dir         string
		args        []string
		expectImage string
		setup       func(t *testing.T, workdir string) (teardown func())
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
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.setup != nil {
				teardown := test.setup(t, test.dir)
				defer teardown()
			}

			// Run without artifact caching
			removeImage(t, test.expectImage)
			skaffold.Build(append(test.args, "--cache-artifacts=false")...).InDir(test.dir).RunOrFail(t)
			checkImageExists(t, test.expectImage)

			// Run with artifact caching
			removeImage(t, test.expectImage)
			skaffold.Build(append(test.args, "--cache-artifacts=true")...).InDir(test.dir).RunOrFail(t)
			checkImageExists(t, test.expectImage)

			// Run a second time with artifact caching
			out := skaffold.Build(append(test.args, "--cache-artifacts=true")...).InDir(test.dir).RunOrFailOutput(t)
			if strings.Contains(string(out), "Not found. Building") {
				t.Errorf("images were expected to be found in cache: %s", out)
			}
			checkImageExists(t, test.expectImage)
		})
	}
}

//see integration/testdata/README.md for details
func TestBuildInCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is gcp only")
	}

	testutil.Run(t, "", func(t *testutil.T) {
		// copy the skaffold binary to the test case folder
		// this is geared towards the in-docker setup: the fresh built binary is here
		// for manual testing, we can override this temporarily
		skaffoldSrc, err := exec.LookPath("skaffold")
		if err != nil {
			t.Fatalf("failed to find skaffold binary: %s", err)
		}
		skaffoldDst := "./testdata/skaffold-in-cluster/skaffold"
		t.CopyFile(skaffoldSrc, skaffoldDst)

		ns, k8sClient, cleanupNs := SetupNamespace(t.T)
		defer cleanupNs()

		// TODO: until https://github.com/GoogleContainerTools/skaffold/issues/2757 is resolved, this is the simplest
		// way to override the build.cluster.namespace
		revert := replaceNamespace("./testdata/skaffold-in-cluster/skaffold.yaml", t, ns)
		defer revert()
		revert = replaceNamespace("./testdata/skaffold-in-cluster/build-step/kustomization.yaml", t, ns)
		defer revert()

		//we have to copy the e2esecret from default ns -> temporary namespace for kaniko
		secret, err := k8sClient.client.CoreV1().Secrets("default").Get("e2esecret", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed reading default/e2escret: %s", err)
		}
		secret.Namespace = ns.Name
		secret.ResourceVersion = ""
		_, err = k8sClient.client.CoreV1().Secrets(ns.Name).Create(secret)
		if err != nil {
			t.Fatalf("failed creating %s/e2escret: %s", ns.Name, err)
		}

		logs := skaffold.Run("-p", "create-build-step", "--cache-artifacts=true").InDir("./testdata/skaffold-in-cluster").InNs(ns.Name).RunOrFailOutput(t.T)
		t.Logf("create-build-step logs: \n%s", logs)

		k8sClient.WaitForPodsInPhase(corev1.PodSucceeded, "skaffold-in-cluster")
	})
}

func replaceNamespace(fileName string, t *testutil.T, ns *corev1.Namespace) func() {
	origSkaffoldYaml, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatalf("failed reading %s: %s", fileName, err)
	}
	namespacedYaml := strings.ReplaceAll(string(origSkaffoldYaml), "VAR_CLUSTER_NAMESPACE", ns.Name)
	if err := ioutil.WriteFile(fileName, []byte(namespacedYaml), 0666); err != nil {
		t.Fatalf("failed to write %s: %s", fileName, err)
	}
	return func() {
		ioutil.WriteFile(fileName, origSkaffoldYaml, 0666)
	}
}

// removeImage removes the given image if present.
func removeImage(t *testing.T, image string) {
	t.Helper()

	if image == "" {
		return
	}

	client, err := docker.NewAPIClient(&runcontext.RunContext{})
	failNowIfError(t, err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	_, _ = client.ImageRemove(ctx, image, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
}

// checkImageExists asserts that the given image is present
func checkImageExists(t *testing.T, image string) {
	t.Helper()

	if image == "" {
		return
	}

	client, err := docker.NewAPIClient(&runcontext.RunContext{})
	failNowIfError(t, err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	if !client.ImageExists(ctx, image) {
		t.Errorf("expected image '%s' not present", image)
	}
}

// setupGitRepo sets up a clean repo with tag v1
func setupGitRepo(t *testing.T, dir string) func() {
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

	return func() {
		os.RemoveAll(dir + "/.git")
	}
}

// nowInChicago returns the dateTime string as generated by the dateTime tagger
func nowInChicago() string {
	loc, _ := tz.LoadLocation("America/Chicago")
	return time.Now().In(loc).Format("2006-01-02")
}

func failNowIfError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

// TestExpectedBuildFailures verifies that `skaffold build` fails in expected ways
func TestExpectedBuildFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

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
