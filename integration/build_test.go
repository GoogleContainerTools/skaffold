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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	kubectx "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const imageName = "us-central1-docker.pkg.dev/k8s-skaffold/testing/simple-build:"

func TestBuild(t *testing.T) {
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
		{
			description: "envTemplate command tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "envTemplateCmd"},
			expectImage: imageName + "1.0.0",
		},
		{
			description: "envTemplate default tagger",
			dir:         "testdata/tagPolicy",
			args:        []string{"-p", "envTemplateDefault"},
			expectImage: imageName + "bar",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
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

func TestBuildWithWithPlatform(t *testing.T) {
	tests := []struct {
		description       string
		dir               string
		args              []string
		image             string
		expectedPlatforms []v1.Platform
	}{
		{
			description:       "docker build linux/amd64",
			dir:               "testdata/build/docker-with-platform-amd",
			args:              []string{"--platform", "linux/amd64"},
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}},
		},
		{
			description:       "docker build linux/arm64",
			dir:               "testdata/build/docker-with-platform-arm",
			args:              []string{"--platform", "linux/arm64"},
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			tmpfile := testutil.TempFile(t, "", []byte{})
			args := append(test.args, "--file-output", tmpfile)
			skaffold.Build(args...).InDir(test.dir).RunOrFail(t)
			bytes, err := os.ReadFile(tmpfile)
			failNowIfError(t, err)
			buildArtifacts, err := flags.ParseBuildOutput(bytes)
			failNowIfError(t, err)
			checkLocalImagePlatforms(t, buildArtifacts.Builds[0].Tag, test.expectedPlatforms)
		})
	}
}

func TestBuildWithMultiPlatforms(t *testing.T) {
	tests := []struct {
		description       string
		dir               string
		args              []string
		image             string
		expectedPlatforms []v1.Platform
	}{
		{
			description:       "build cross platform images with gcb",
			dir:               "testdata/build/gcb-with-platform",
			args:              []string{"--platform", "linux/arm64,linux/amd64"},
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, NeedsGcp)
			tmpfile := testutil.TempFile(t, "", []byte{})
			args := append(test.args, "--file-output", tmpfile)
			skaffold.Build(args...).InDir(test.dir).RunOrFail(t)
			bytes, err := os.ReadFile(tmpfile)
			failNowIfError(t, err)
			buildArtifacts, err := flags.ParseBuildOutput(bytes)
			failNowIfError(t, err)
			checkRemoteImagePlatforms(t, buildArtifacts.Builds[0].Tag, test.expectedPlatforms)
		})
	}
}

// TestExpectedBuildFailures verifies that `skaffold build` fails in expected ways
func TestExpectedBuildFailures(t *testing.T) {
	if !jib.JVMFound(context.Background()) {
		t.Fatal("test requires Java VM")
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
			MarkIntegrationTest(t, NeedsGcp)
			if out, err := skaffold.Build(test.args...).InDir(test.dir).RunWithCombinedOutput(t); err == nil {
				t.Fatal("expected build to fail")
			} else if !strings.Contains(string(out), test.expected) {
				t.Log("build output: ", string(out))
				t.Fatalf("build failed but for wrong reason")
			}
		})
	}
}

func checkLocalImagePlatforms(t *testing.T, image string, expected []v1.Platform) {
	if expected == nil {
		return
	}
	t.Helper()

	cfg, err := kubectx.CurrentConfig()
	failNowIfError(t, err)

	client, err := docker.NewAPIClient(context.Background(), &runcontext.RunContext{
		KubeContext: cfg.CurrentContext,
	})
	failNowIfError(t, err)
	inspect, _, err := client.ImageInspectWithRaw(context.Background(), image)
	failNowIfError(t, err)

	actual := []v1.Platform{{Architecture: inspect.Architecture, OS: inspect.Os}}
	checkPlatformsEqual(t, actual, expected)
}

func checkRemoteImagePlatforms(t *testing.T, image string, expected []v1.Platform) {
	if expected == nil {
		return
	}
	t.Helper()
	actual, err := docker.GetPlatforms(image)
	if err != nil {
		t.Error(err)
	}
	checkPlatformsEqual(t, actual, expected)
}

func checkPlatformsEqual(t *testing.T, actual, expected []v1.Platform) {
	platLess := func(a, b v1.Platform) bool {
		return a.OS < b.OS || (a.OS == b.OS && a.Architecture < b.Architecture)
	}
	if diff := cmp.Diff(expected, actual, cmpopts.SortSlices(platLess)); diff != "" {
		t.Fatalf("Platforms differ (-got,+want):\n%s", diff)
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
	client, err := docker.NewAPIClient(context.Background(), &runcontext.RunContext{
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
		if buf, err := util.RunCmdOut(context.Background(), cmd); err != nil {
			t.Log(string(buf))
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

func TestRunWithDockerAndBuildArgs(t *testing.T) {
	// Skip in hybrid environment
	if os.Getenv("GKE_CLUSTER_NAME") == "presubmit-hybrid" {
		t.Skip("Skipping test in hybrid environment: docker-container driver stores images in BuildKit cache, not local daemon")
	}
	tests := []struct {
		description   string
		projectDir    string
		skaffoldArgs  []string
		dockerRunArgs []string
		wantOutput    string
	}{
		{
			description:   "IMAGE_REPO, IMAGE_TAG, and IMAGE_NAME are passed to Docker build as build args",
			projectDir:    "testdata/docker-run-with-build-args/artifact-with-dependency",
			skaffoldArgs:  []string{"--kube-context", "default"},
			dockerRunArgs: []string{"run", "child:latest"},
			wantOutput:    "IMAGE_REPO: gcr.io/k8s-skaffold, IMAGE_NAME: skaffold, IMAGE_TAG:latest",
		},
		{
			description:   "IMAGE_TAG can be used as a part of a filename in the Dockerfile",
			projectDir:    "testdata/docker-run-with-build-args/single-artifact",
			skaffoldArgs:  []string{"--kube-context", "default"},
			dockerRunArgs: []string{"run", "example:latest"},
			wantOutput:    "HELLO WORLD",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			defer skaffold.Delete().InDir(test.projectDir).Run(t.T)
			if err := skaffold.Build(test.skaffoldArgs...).InDir(test.projectDir).Run(t.T); err != nil {
				t.Errorf("skaffold build args: %v  working directory:%s returned unexpected error: %v", test.skaffoldArgs, test.projectDir, err)
			}

			got := ""

			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				out, _ := exec.Command("docker", test.dockerRunArgs...).Output()
				t.Logf("Output:[%s]\n", out)
				got = strings.Trim(string(out), " \n")
				return got == test.wantOutput, nil
			})

			if err != nil {
				t.Errorf("docker run produced incorrect output, got:[%s], want:[%s], err: %v", got, test.wantOutput, err)
			}
			failNowIfError(t, err)
		})
	}
}
