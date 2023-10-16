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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		args []string
	}{
		/*
			// Fix after https://github.com/GoogleContainerTools/skaffold/issues/6722
				{
					name: "compose",
					dir:  "testdata/init/compose",
					args: []string{"--compose-file", "docker-compose.yaml"},
				},
		*/
		{
			name: "helm init",
			dir:  "testdata/init/helm-project",
			args: []string{"--XXenableBuildpacksInit=false"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			ns, _ := SetupNamespace(t.T)

			initArgs := append([]string{"--force"}, test.args...)
			skaffold.Init(initArgs...).InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)
			checkGeneratedConfig(t, test.dir)

			// Make sure the skaffold yaml and the kubernetes manifests created by kompose are ok
			skaffold.Run().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
			defer skaffold.Delete().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name)
		})
	}
}

func TestInitManifestGeneration(t *testing.T) {
	t.Skipf("Fix after https://github.com/GoogleContainerTools/skaffold/issues/6722")

	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		name                  string
		dir                   string
		args                  []string
		expectedManifestPaths []string
	}{
		{
			name:                  "hello",
			dir:                   "testdata/init/hello",
			args:                  []string{"--generate-manifests"},
			expectedManifestPaths: []string{"deployment.yaml"},
		},
		// TODO(nkubala): add this back when the --force flag is fixed
		// {
		// 	name:                  "microservices",
		// 	dir:                   "testdata/init/microservices",
		// 	args:                  []string{"--XXenableManifestGeneration"},
		// 	expectedManifestPaths: []string{"leeroy-web/deployment.yaml", "leeroy-app/deployment.yaml"},
		// },
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			ns, _ := SetupNamespace(t.T)

			initArgs := append([]string{"--force"}, test.args...)
			skaffold.Init(initArgs...).InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

			checkGeneratedManifests(t, test.dir, test.expectedManifestPaths)

			// Make sure the skaffold yaml and the kubernetes manifests created by kompose are ok
			skaffold.Run().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
			skaffold.Delete().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
		})
	}
}

func TestInitKustomize(t *testing.T) {
	t.Skipf("Fix after https://github.com/GoogleContainerTools/skaffold/issues/6722")

	MarkIntegrationTest(t, CanRunWithoutGcp)

	testutil.Run(t, "kustomize init", func(t *testutil.T) {
		dir := "examples/getting-started-kustomize"
		ns, _ := SetupNamespace(t.T)

		initArgs := []string{"--force"}
		defer func() {
			path := filepath.Join(dir, "skaffold.yaml.out")
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				return
			}
			os.Remove(path)
		}()
		skaffold.Init(initArgs...).InDir(dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

		checkGeneratedConfig(t, dir)

		skaffold.Run().InDir(dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
		skaffold.Delete().InDir(dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
	})
}

func TestInitWithCLIArtifact(t *testing.T) {
	t.Skipf("Fix after https://github.com/GoogleContainerTools/skaffold/issues/6722")

	MarkIntegrationTest(t, CanRunWithoutGcp)

	testutil.Run(t, "init with cli artifact", func(t *testutil.T) {
		dir := "testdata/init/hello-with-manifest"
		ns, _ := SetupNamespace(t.T)

		initArgs := append([]string{"--force"},
			`--artifact={"builder":"Docker","payload":{"path":"../hello/Dockerfile"},"image":"dockerfile-image"}`)
		skaffold.Init(initArgs...).InDir(dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

		checkGeneratedConfig(t, dir)

		// Make sure the skaffold yaml is ok
		skaffold.Run().InDir(dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
		skaffold.Delete().InDir(dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
	})
}

func TestInitWithCLIArtifactAndManifestGeneration(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	t.Skipf("Fix after https://github.com/GoogleContainerTools/skaffold/issues/6722")

	testutil.Run(t, "init with cli artifact and manifests", func(t *testutil.T) {
		ns, _ := SetupNamespace(t.T)
		dir := "testdata/init/hello"

		initArgs := append([]string{"--force"},
			`--artifact={"builder":"Docker","payload":{"path":"./Dockerfile"},"image":"dockerfile-image","manifest":{"generate":true,"port":8080}}`)
		skaffold.Init(initArgs...).InDir(dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

		checkGeneratedManifests(t, dir, []string{"deployment.yaml"})

		skaffold.Run().InDir(dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
	})
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := os.ReadFile(filepath.Join(dir, "skaffold.yaml"))
	t.CheckNoError(err)

	output, err := os.ReadFile(filepath.Join(dir, "skaffold.yaml.out"))
	t.CheckNoError(err)
	t.CheckDeepEqual(string(expectedOutput), string(output), testutil.YamlObj(t.T))
}

func checkGeneratedManifests(t *testutil.T, dir string, manifestPaths []string) {
	for _, path := range manifestPaths {
		expectedOutput, err := os.ReadFile(filepath.Join(dir, strings.Join([]string{".", path, ".expected"}, "")))
		t.CheckNoError(err)

		output, err := os.ReadFile(filepath.Join(dir, path))
		t.CheckNoError(err)
		t.CheckDeepEqual(string(expectedOutput), string(output))
	}
}

func TestInitFailures(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	testutil.Run(t, "no builder", func(t *testutil.T) {
		out, err := skaffold.Init().InDir("testdata/init/no-builder").RunWithCombinedOutput(t.T)

		t.CheckContains("please provide at least one build config", string(out))
		t.CheckDeepEqual(101, exitCode(err))
	})
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if ok := errors.As(err, &exitErr); ok {
		return exitErr.ExitCode()
	}

	return 1
}
