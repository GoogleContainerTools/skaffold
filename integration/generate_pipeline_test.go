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
	"bytes"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

type configContents struct {
	path string
	data []byte
}

func TestGeneratePipeline(t *testing.T) {
	// TODO: This test shall pass once render v2 is completed.
	t.SkipNow()

	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		dir         string
		input       []byte
		args        []string
		configFiles []string
		skipCheck   bool
	}{
		{
			description: "no profiles",
			dir:         "testdata/generate_pipeline/no_profiles",
			input:       []byte("y\n"),
		},
		{
			description: "existing oncluster profile",
			dir:         "testdata/generate_pipeline/existing_oncluster",
			input:       []byte(""),
		},
		{
			description: "existing other profile",
			dir:         "testdata/generate_pipeline/existing_other",
			input:       []byte("y\n"),
		},
		{
			description: "multiple skaffold.yamls to create pipeline from",
			dir:         "testdata/generate_pipeline/multiple_configs",
			input:       []byte{'y', '\n', 'y', '\n'},
			configFiles: []string{"sub-app/skaffold.yaml"},
		},
		{
			description: "user example",
			dir:         "examples/generate-pipeline",
			input:       []byte("y\n"),
			skipCheck:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			args, contents, err := getOriginalContents(test.args, test.dir, test.configFiles)
			failNowIfError(t, err)
			defer writeOriginalContents(contents)

			originalConfig, err := os.ReadFile(test.dir + "/skaffold.yaml")
			failNowIfError(t, err)
			defer os.WriteFile(test.dir+"/skaffold.yaml", originalConfig, 0755)
			defer os.Remove(test.dir + "/pipeline.yaml")

			skaffoldEnv := []string{
				"PIPELINE_GIT_URL=this-is-a-test",
				"PIPELINE_SKAFFOLD_VERSION=latest",
			}
			skaffold.GeneratePipeline(args...).WithStdin(test.input).WithEnv(skaffoldEnv).InDir(test.dir).RunOrFail(t)

			if !test.skipCheck {
				checkFileContents(t, test.dir+"/expectedSkaffold.yaml", test.dir+"/skaffold.yaml")
			}
			checkFileContents(t, test.dir+"/expectedPipeline.yaml", test.dir+"/pipeline.yaml")
		})
	}
}

func getOriginalContents(testArgs []string, testDir string, configFiles []string) ([]string, []configContents, error) {
	var originalConfigs []configContents

	for _, configFile := range configFiles {
		testArgs = append(testArgs, []string{"--config-files", configFile}...)

		path := testDir + "/" + configFile
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		originalConfigs = append(originalConfigs, configContents{path, contents})
	}

	return testArgs, originalConfigs, nil
}

func writeOriginalContents(contents []configContents) {
	for _, content := range contents {
		os.WriteFile(content.path, content.data, 0755)
	}
}

func checkFileContents(t *testing.T, wantFile, gotFile string) {
	wantContents, err := os.ReadFile(wantFile)
	failNowIfError(t, err)

	gotContents, err := os.ReadFile(gotFile)
	failNowIfError(t, err)

	if !bytes.Equal(wantContents, gotContents) {
		t.Errorf("Contents of %s did not match those of %s\ngot:%s\nwant:%s", gotFile, wantFile, string(gotContents), string(wantContents))
	}
}
