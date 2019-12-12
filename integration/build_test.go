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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"4d63.com/tz"
	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/kubernetes"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const imageName = "simple-build:"

func TestBuild(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
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
		{
			description: "custom",
			dir:         "examples/custom",
			setup: func(t *testing.T, _ string) func() {
				cmd := exec.Command("pack", "set-default-builder", "heroku/buildpacks")
				if err := cmd.Run(); err != nil {
					t.Fatalf("error setting default buildpacks builder: %v", err)
				}
				return func() {}
			},
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
	if testing.Short() || !RunOnGCP() {
		t.Skip("skipping GCP integration test")
	}

	testutil.Run(t, "", func(t *testutil.T) {
		ns, k8sClient, cleanupNs := SetupNamespace(t.T)
		defer cleanupNs()

		// this workaround is to ensure there is no overlap between testcases on kokoro
		// see https://github.com/GoogleContainerTools/skaffold/issues/2781#issuecomment-527770537
		project, err := filepath.Abs("testdata/skaffold-in-cluster")
		if err != nil {
			t.Fatalf("failed getting path to project: %s", err)
		}

		// copy the skaffold binary to the test case folder
		// this is geared towards the in-docker setup: the fresh built binary is here
		// for manual testing, we can override this temporarily
		skaffoldSrc, err := exec.LookPath("skaffold")
		if err != nil {
			t.Fatalf("failed to find skaffold binary: %s", err)
		}

		t.NewTempDir().Chdir()
		copyDir(t, project, ".")
		copyFile(t, skaffoldSrc, "skaffold")

		// TODO: until https://github.com/GoogleContainerTools/skaffold/issues/2757 is resolved, this is the simplest
		// way to override the build.cluster.namespace
		replaceNamespace(t, "skaffold.yaml", ns)
		replaceNamespace(t, "build-step/kustomization.yaml", ns)

		// we have to copy the e2esecret from default ns -> temporary namespace for kaniko
		secret, err := k8sClient.client.CoreV1().Secrets("default").Get("e2esecret", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed reading default/e2esecret: %s", err)
		}
		secret.Namespace = ns.Name
		secret.ResourceVersion = ""
		if _, err = k8sClient.Secrets().Create(secret); err != nil {
			t.Fatalf("failed creating %s/e2esecret: %s", ns.Name, err)
		}

		logs := skaffold.Run("-p", "create-build-step").InNs(ns.Name).RunOrFailOutput(t.T)
		t.Logf("create-build-step logs: \n%s", logs)

		k8sClient.WaitForPodsInPhase(corev1.PodSucceeded, "skaffold-in-cluster")
	})
}

func replaceNamespace(t *testutil.T, fileName string, ns *corev1.Namespace) {
	origSkaffoldYaml, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatalf("failed reading %s: %s", fileName, err)
	}

	namespacedYaml := strings.ReplaceAll(string(origSkaffoldYaml), "VAR_CLUSTER_NAMESPACE", ns.Name)

	if err := ioutil.WriteFile(fileName, []byte(namespacedYaml), 0666); err != nil {
		t.Fatalf("failed to write %s: %s", fileName, err)
	}
}

func copyFile(t *testutil.T, src, dst string) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		t.Fatalf("can't read source file: %s: %s", src, err)
	}

	err = ioutil.WriteFile(dst, content, 0666)
	if err != nil {
		t.Fatalf("failed to copy file %s to %s: %s", src, dst, err)
	}
}

func copyDir(t *testutil.T, src string, dst string) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		t.Fatalf("failed to copy dir %s->%s: %s ", src, dst, err)
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		t.Fatalf("failed to copy dir %s->%s: %s ", src, dst, err)
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to copy dir %s->%s: %s ", src, dst, err)
	}

	for _, f := range files {
		srcfp := path.Join(src, f.Name())
		dstfp := path.Join(dst, f.Name())

		if f.IsDir() {
			copyDir(t, srcfp, dstfp)
		} else {
			copyFile(t, srcfp, dstfp)
		}
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
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
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

// run on GCP as this test requires a load balancer
func TestBuildKanikoInsecureRegistry(t *testing.T) {
	if testing.Short() || !RunOnGCP() {
		t.Skip("skipping GCP integration test")
	}

	ns, k8sClient, cleanupNs := SetupNamespace(t)
	defer cleanupNs()

	dir := "testdata/kaniko-insecure-registry"

	cleanup := deployInsecureRegistry(t, ns.Name, dir)
	defer cleanup()

	ip := getExternalIP(t, k8sClient, ns.Name)
	registry := fmt.Sprintf("%s:5000", ip)

	skaffold.Build("--insecure-registry", registry, "-d", registry, "-p", "build-artifact").InDir(dir).InNs(ns.Name).RunOrFailOutput(t)
}

func deployInsecureRegistry(t *testing.T, ns, dir string) func() {
	skaffold.Run("-p", "deploy-insecure-registry").InDir(dir).InNs(ns).RunOrFailOutput(t)

	cleanup := func() {
		skaffold.Delete("-p", "deploy-insecure-registry").InDir(dir).InNs(ns).RunOrFailOutput(t)
	}
	return cleanup
}

func getExternalIP(t *testing.T, c *NSKubernetesClient, ns string) string {
	svc, err := c.client.CoreV1().Services(ns).Get("registry", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting registry service: %v", err)
	}
	// Wait for external IP of service
	ip, err := kubernetes.GetExternalIP(svc)
	if err != nil {
		t.Fatalf("error getting external ip: %v", err)
	}
	return ip
}

func TestBuildGCBWithDefaultRepo(t *testing.T) {
	if testing.Short() || !RunOnGCP() {
		t.Skip("skipping GCP integration test")
	}

	// The GCB project (k8s-skaffold) has to be deduced from artifact's image name
	// after the default repo is applied.
	// If it's not properly resolved, the build will fail.
	skaffold.Build("-d", "gcr.io/k8s-skaffold").InDir("testdata/gcb-default-repo").RunOrFail(t)
}

func TestBuildKanikoWithDefaultRepo(t *testing.T) {
	if testing.Short() || !RunOnGCP() {
		t.Skip("skipping GCP integration test")
	}

	// The GCS project (k8s-skaffold) has to be deduced from artifact's image name
	// after the default repo is applied.
	// If it's not properly resolved, the build will fail.
	skaffold.Build("-d", "gcr.io/k8s-skaffold").InDir("testdata/kaniko-default-repo").RunOrFail(t)
}
