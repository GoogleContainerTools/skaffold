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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInitCompose(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		name string
		dir  string
		args []string
	}{
		{
			name: "compose",
			dir:  "testdata/init/compose",
			args: []string{"--compose-file", "docker-compose.yaml"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			ns, _ := SetupNamespace(t.T)

			initArgs := append([]string{"--force"}, test.args...)
			skaffold.Init(initArgs...).InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

			checkGeneratedConfig(t, test.dir)

			// Make sure the skaffold yaml and the kubernetes manifests created by kompose are ok
			skaffold.Run().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
		})
	}
}

func TestInitManifestGeneration(t *testing.T) {
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
			args:                  []string{"--XXenableManifestGeneration"},
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
		})
	}
}

func TestInitKustomize(t *testing.T) {
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
	})
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := ioutil.ReadFile(filepath.Join(dir, "skaffold.yaml"))
	t.CheckNoError(err)

	output, err := ioutil.ReadFile(filepath.Join(dir, "skaffold.yaml.out"))
	t.CheckNoError(err)
	t.CheckDeepEqual(string(expectedOutput), string(output))
}

func checkGeneratedManifests(t *testutil.T, dir string, manifestPaths []string) {
	for _, path := range manifestPaths {
		expectedOutput, err := ioutil.ReadFile(filepath.Join(dir, path+".expected"))
		t.CheckNoError(err)

		output, err := ioutil.ReadFile(filepath.Join(dir, path))
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
