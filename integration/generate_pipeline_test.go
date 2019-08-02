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
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestGeneratePipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	tests := []struct {
		description string
		dir         string
		responses   []byte
	}{
		{
			description: "no profiles",
			dir:         "testdata/generate_pipeline/no_profiles",
			responses:   []byte("y"),
		},
		{
			description: "existing oncluster profile",
			dir:         "testdata/generate_pipeline/existing_oncluster",
			responses:   []byte(""),
		},
		{
			description: "existing other profile",
			dir:         "testdata/generate_pipeline/existing_other",
			responses:   []byte("y"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			originalConfig, err := ioutil.ReadFile(test.dir + "/skaffold.yaml")
			if err != nil {
				t.Error("error reading skaffold yaml")
			}
			defer ioutil.WriteFile(test.dir+"/skaffold.yaml", originalConfig, 0755)

			skaffoldEnv := []string{
				"PIPELINE_GIT_URL=this-is-a-test",
				"PIPELINE_SKAFFOLD_VERSION=test-version",
			}
			skaffold.GeneratePipeline().WithStdin([]byte("y\n")).WithEnv(skaffoldEnv).InDir(test.dir).RunOrFail(t)

			checkFileContents(t, test.dir+"/expectedSkaffold.yaml", test.dir+"/skaffold.yaml")
			checkFileContents(t, test.dir+"/expectedPipeline.yaml", test.dir+"/pipeline.yaml")
		})
	}
}

func checkFileContents(t *testing.T, wantFile, gotFile string) {
	wantContents, err := ioutil.ReadFile(wantFile)
	if err != nil {
		t.Errorf("Error while reading contents of file %s", wantFile)
	}
	gotContents, err := ioutil.ReadFile(gotFile)
	if err != nil {
		t.Errorf("Error while reading contents of file %s", gotFile)
	}

	if !bytes.Equal(wantContents, gotContents) {
		t.Errorf("Contents of %s did not match those of %s\ngot:%s\nwant:%s", gotFile, wantFile, string(gotContents), string(wantContents))
	}
}
