/*
Copyright 2021 The Skaffold Authors

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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInspectBuildEnv(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	gcbParams := []string{
		"--projectId", "proj2",
		"--workerPool", "pool2",
		"--timeout", "180s",
		"--machineType", "vm2",
		"--logStreamingOption", "STREAM_ON",
		"--logging", "LEGACY",
		"--diskSizeGb", "10",
		"--concurrency", "2",
	}
	tests := []struct {
		description        string
		inputConfigFile    string
		args               []string
		expectedConfigFile string
		expectedOut        string
	}{
		{
			description:        "add new gcb build env definition in default pipeline",
			inputConfigFile:    "skaffold.local.yaml",
			expectedConfigFile: "skaffold.gcb.add.default.yaml",
			args:               append([]string{"build-env", "add", "googleCloudBuild"}, gcbParams...),
		},
		{
			description:        "add new gcb build env definition in new profile",
			inputConfigFile:    "skaffold.gcb.yaml",
			expectedConfigFile: "skaffold.gcb.add.profile.yaml",
			args:               append([]string{"build-env", "add", "googleCloudBuild", "--profile", "gcb"}, gcbParams...),
		},
		{
			description:        "modify existing gcb build env definition in default pipeline",
			inputConfigFile:    "skaffold.gcb.yaml",
			expectedConfigFile: "skaffold.gcb.modified.default.yaml",
			args:               append([]string{"build-env", "modify", "googleCloudBuild"}, gcbParams...),
		},
		{
			description:        "modify existing gcb build env definition in existing profile",
			inputConfigFile:    "skaffold.local.yaml",
			expectedConfigFile: "skaffold.gcb.modified.profile.yaml",
			args:               append([]string{"build-env", "modify", "googleCloudBuild", "--profile", "gcb"}, gcbParams...),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			configContents, err := ioutil.ReadFile(filepath.Join("testdata/inspect", test.inputConfigFile))
			t.CheckNoError(err)
			tmpDir.Write("skaffold.yaml", string(configContents))
			args := append(test.args, fmt.Sprintf("-f=%s", tmpDir.Path("skaffold.yaml")))
			out := skaffold.Inspect(args...).InDir(tmpDir.Root()).RunOrFailOutput(t.T)
			if test.expectedOut != "" {
				t.CheckDeepEqual(test.expectedOut, string(out))
			}
			if test.expectedConfigFile != "" {
				expectedConfig, err := ioutil.ReadFile(filepath.Join("testdata/inspect", test.expectedConfigFile))
				t.CheckNoError(err)
				actualConfig, err := ioutil.ReadFile(tmpDir.Path("skaffold.yaml"))
				t.CheckNoError(err)
				t.CheckDeepEqual(expectedConfig, actualConfig)
			}
		})
	}
}
