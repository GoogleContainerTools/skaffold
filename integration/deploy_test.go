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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestBuildDeploy(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	ns, client := SetupNamespace(t)

	outputBytes := skaffold.Build("--quiet", "--platform=linux/arm64,linux/amd64").InDir("examples/nodejs").InNs(ns.Name).RunOrFailOutput(t)
	// Parse the Build Output
	buildArtifacts, err := flags.ParseBuildOutput(outputBytes)
	failNowIfError(t, err)

	tmpDir := testutil.NewTempDir(t)
	buildOutputFile := tmpDir.Path("build.out")
	tmpDir.Write("build.out", string(outputBytes))

	// Run Deploy using the build output
	// See https://github.com/GoogleContainerTools/skaffold/issues/2372 on why status-check=false
	skaffold.Deploy("--build-artifacts", buildOutputFile, "--status-check=false").InDir("examples/nodejs").InNs(ns.Name).RunOrFail(t)

	depApp := client.GetDeployment("node")
	testutil.CheckDeepEqual(t, buildArtifacts.Builds[0].Tag, depApp.Spec.Template.Spec.Containers[0].Image)
}

func TestDeploy(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, client := SetupNamespace(t)

	// `--default-repo=` is used to cancel the default repo that is set by default.
	skaffold.Deploy("--images", "index.docker.io/library/busybox:1", "--default-repo=").InDir("examples/kustomize").InNs(ns.Name).RunOrFail(t)

	dep := client.GetDeployment("kustomize-test")
	testutil.CheckDeepEqual(t, "index.docker.io/library/busybox:1", dep.Spec.Template.Spec.Containers[0].Image)
}

func TestDeployWithBuildArtifacts(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, client := SetupNamespace(t)

	// first build the artifacts and output to file
	skaffold.Build("--file-output=images.json", "--default-repo=").InDir("examples/getting-started").RunOrFail(t)

	// `--default-repo=` is used to cancel the default repo that is set by default.
	skaffold.Deploy("--build-artifacts=images.json", "--default-repo=", "--load-images=true").InDir("examples/getting-started").InNs(ns.Name).RunOrFail(t)

	pod := client.GetPod("getting-started")
	testutil.CheckContains(t, "skaffold-example", pod.Spec.Containers[0].Image)
}

func TestDeployWithImages(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, client := SetupNamespace(t)

	// first build the artifacts and output to file
	skaffold.Build("--file-output=artifacts.json", "--default-repo=").InDir("examples/getting-started").RunOrFail(t)

	var artifacts flags.BuildOutput
	if ba, err := os.ReadFile("examples/getting-started/artifacts.json"); err != nil {
		t.Fatal("could not read artifacts.json", err)
	} else if err := json.Unmarshal(ba, &artifacts); err != nil {
		t.Fatal("could not decode artifacts.json", err)
	}

	var images []string
	for _, a := range artifacts.Builds {
		images = append(images, a.ImageName+"="+a.Tag)
	}

	// `--default-repo=` is used to cancel the default repo that is set by default.
	skaffold.Deploy("--images="+strings.Join(images, ","), "--default-repo=", "--load-images=true").InDir("examples/getting-started").InNs(ns.Name).RunOrFail(t)

	pod := client.GetPod("getting-started")
	testutil.CheckContains(t, "skaffold-example", pod.Spec.Containers[0].Image)
}

func TestDeployTail(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	// `--default-repo=` is used to cancel the default repo that is set by default.
	out := skaffold.Deploy("--tail", "--images", "busybox:latest", "--default-repo=").InDir("testdata/deploy-hello-tail").InNs(ns.Name).RunLive(t)

	WaitForLogs(t, out, "Hello world!")
}

func TestDeployTailDefaultNamespace(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// `--default-repo=` is used to cancel the default repo that is set by default.
	out := skaffold.Deploy("--tail", "--images", "busybox:latest", "--default-repo=").InDir("testdata/deploy-hello-tail").RunLive(t)

	defer skaffold.Delete().InDir("testdata/deploy-hello-tail").Run(t)
	WaitForLogs(t, out, "Hello world!")
}

func TestDeployWithInCorrectConfig(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	// We're not providing a tag for the getting-started image
	output, err := skaffold.Deploy().InDir("examples/getting-started").InNs(ns.Name).RunWithCombinedOutput(t)
	if err == nil {
		t.Errorf("expected to see an error since not every image tag is provided: %s", output)
	} else if !strings.Contains(string(output), "no tag provided for image [skaffold-example]") {
		t.Errorf("failed without saying the reason: %s", output)
	}
}

// Verify that we can deploy without artifact details (https://github.com/GoogleContainerTools/skaffold/issues/4616)
func TestDeployWithoutWorkspaces(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	ns, _ := SetupNamespace(t)

	outputBytes := skaffold.Build("--quiet").InDir("examples/nodejs").InNs(ns.Name).RunOrFailOutput(t)
	// Parse the Build Output
	buildArtifacts, err := flags.ParseBuildOutput(outputBytes)
	failNowIfError(t, err)
	if len(buildArtifacts.Builds) != 1 {
		t.Fatalf("expected 1 artifact to be built, but found %d", len(buildArtifacts.Builds))
	}

	tmpDir := testutil.NewTempDir(t)
	buildOutputFile := tmpDir.Path("build.out")
	tmpDir.Write("build.out", string(outputBytes))
	copyFiles(tmpDir.Root(), "examples/nodejs/skaffold.yaml")
	copyFiles(tmpDir.Root(), "examples/nodejs/k8s")

	// Run Deploy using the build output
	// See https://github.com/GoogleContainerTools/skaffold/issues/2372 on why status-check=false
	skaffold.Deploy("--build-artifacts", buildOutputFile, "--status-check=false").InDir(tmpDir.Root()).InNs(ns.Name).RunOrFail(t)
}

func TestDeployDependenciesOrder(t *testing.T) {
	tests := []struct {
		description         string
		dir                 string
		moduleToDeploy      string
		expectedDeployOrder []string
	}{
		{
			description: "Deploy order of entire project",
			dir:         "testdata/multi-config-dependencies-order",
			expectedDeployOrder: []string{
				"module4",
				"module3",
				"module2",
				"module1",
			},
		},
		{
			description:    "Deploy order for just one part of the project",
			dir:            "testdata/multi-config-dependencies-order",
			moduleToDeploy: "module2",
			expectedDeployOrder: []string{
				"module4",
				"module3",
				"module2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			targetModule := []string{}
			if test.moduleToDeploy != "" {
				targetModule = []string{"--module", test.moduleToDeploy}
			}

			expectedFormatedDeployOrder := []string{}
			for _, module := range test.expectedDeployOrder {
				expectedFormatedDeployOrder = append(expectedFormatedDeployOrder, fmt.Sprintf(" - pod/%v created", module))
				expectedFormatedDeployOrder = append(expectedFormatedDeployOrder, "Waiting for deployments to stabilize...")
				expectedFormatedDeployOrder = append(expectedFormatedDeployOrder, "Deployments stabilized in \\d*\\.?\\d+ms")
			}
			expectedFormatedDeployOrder = append([]string{"Starting deploy..."}, expectedFormatedDeployOrder...)
			expectedOutput := strings.Join(expectedFormatedDeployOrder, "\n")

			ns, _ := SetupNamespace(t)
			outputBytes := skaffold.Run(targetModule...).InDir(test.dir).InNs(ns.Name).RunOrFailOutput(t)
			defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).RunOrFail(t)

			output := string(outputBytes)
			testutil.CheckRegex(t, expectedOutput, output)
		})
	}
}

// Copies a file or directory tree.  There are 2x3 cases:
//   1. If _src_ is a file,
//      1. and _dst_ exists and is a file then _src_ is copied into _dst_
//      2. and _dst_ exists and is a directory, then _src_ is copied as _dst/$(basename src)_
//      3. and _dst_ does not exist, then _src_ is copied as _dst_.
//   2. If _src_ is a directory,
//      1. and _dst_ exists and is a file, then return an error
//      2. and _dst_ exists and is a directory, then src is copied as _dst/$(basename src)_
//      3. and _dst_ does not exist, then src is copied as _dst/src[1:]_.

func copyFiles(dst, src string) error {
	if util.IsFile(src) {
		switch {
		case util.IsFile(dst): // copy _src_ to _dst_
		case util.IsDir(dst): // copy _src_ to _dst/src[-1]
			dst = filepath.Join(dst, filepath.Base(src))
		default: // copy _src_ to _dst_
			if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
				return err
			}
		}
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		return err
	} else if !util.IsDir(src) {
		return errors.New("src does not exist")
	}
	// so src is a directory
	if util.IsFile(dst) {
		return errors.New("cannot copy directory into file")
	}
	srcPrefix := src
	if util.IsDir(dst) { // src is copied to _dst/$(basename src)
		srcPrefix = filepath.Dir(src)
	} else if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}
	return walk.From(src).Unsorted().WhenIsFile().Do(func(path string, _ walk.Dirent) error {
		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		destFile := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(destFile), os.ModePerm); err != nil {
			return err
		}

		out, err := os.Create(destFile)
		if err != nil {
			return err
		}

		_, err = io.Copy(out, in)
		return err
	})
}
