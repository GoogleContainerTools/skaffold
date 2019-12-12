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
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

type configContents struct {
	path string
	data []byte
}

func TestGeneratePipeline(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

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
			if err != nil {
				t.Fatal(err)
			}
			defer writeOriginalContents(contents)

			originalConfig, err := ioutil.ReadFile(test.dir + "/skaffold.yaml")
			if err != nil {
				t.Error("error reading skaffold yaml")
			}
			defer ioutil.WriteFile(test.dir+"/skaffold.yaml", originalConfig, 0755)
			defer os.Remove(test.dir + "/pipeline.yaml")

			skaffoldEnv := []string{
				"PIPELINE_GIT_URL=this-is-a-test",
				"PIPELINE_SKAFFOLD_VERSION=test-version",
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
	if len(configFiles) != 0 {
		for _, configFile := range configFiles {
			testArgs = append(testArgs, []string{"--config-files", configFile}...)

			path := testDir + "/" + configFile
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, nil, err
			}
			originalConfigs = append(originalConfigs, configContents{path, contents})
		}
	}

	return testArgs, originalConfigs, nil
}

func writeOriginalContents(contents []configContents) {
	for _, content := range contents {
		ioutil.WriteFile(content.path, content.data, 0755)
	}
}

func checkFileContents(t *testing.T, wantFile, gotFile string) {
	wantContents, err := ioutil.ReadFile(wantFile)
	if err != nil {
		t.Errorf("Error while reading contents of file %s: %s", wantFile, err)
	}
	gotContents, err := ioutil.ReadFile(gotFile)
	if err != nil {
		t.Errorf("Error while reading contents of file %s: %s", gotFile, err)
	}

	if !bytes.Equal(wantContents, gotContents) {
		t.Errorf("Contents of %s did not match those of %s\ngot:%s\nwant:%s", gotFile, wantFile, string(gotContents), string(wantContents))
	}
}
