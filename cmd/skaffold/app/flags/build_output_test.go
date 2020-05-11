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

package flags

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewBuildOutputFlag(t *testing.T) {
	flag := NewBuildOutputFileFlag("test.in")
	expectedFlag := &BuildOutputFileFlag{filename: "test.in"}
	if flag.String() != expectedFlag.String() {
		t.Errorf("expected %s, actual %s", expectedFlag, flag)
	}
}

func TestBuildOutputSet(t *testing.T) {
	tests := []struct {
		description         string
		files               map[string]string
		shouldErr           bool
		expectedBuildOutput BuildOutput
	}{
		{
			description: "set returns correct build output format for json",
			files: map[string]string{
				"test.in": `{
"builds": [{
	"imageName": "gcr.io/k8s/test1",
	"tag": "sha256@foo"
	}, {
	"imageName": "gcr.io/k8s/test2",
	"tag": "sha256@bar"
  }]
}`,
			},
			expectedBuildOutput: BuildOutput{
				Builds: []build.Artifact{{
					ImageName: "gcr.io/k8s/test1",
					Tag:       "sha256@foo",
				}, {
					ImageName: "gcr.io/k8s/test2",
					Tag:       "sha256@bar",
				}},
			},
		},
		{
			description: "set errors with in-correct build output format",
			files: map[string]string{
				"test.in": "",
			},
			shouldErr: true,
		},
		{
			description: "set should error when file is not present with original flag value",
			files:       map[string]string{},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				WriteFiles(test.files)

			flag := NewBuildOutputFileFlag("")
			err := flag.Set(tmpDir.Path("test.in"))

			expectedFlag := &BuildOutputFileFlag{
				filename:    "test.in",
				buildOutput: test.expectedBuildOutput,
			}
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expectedFlag.buildOutput, flag.buildOutput)
		})
	}
}

func TestBuildOutputString(t *testing.T) {
	flag := NewBuildOutputFileFlag("test.in")

	testutil.CheckDeepEqual(t, "test.in", flag.String())
	testutil.CheckDeepEqual(t, "*flags.BuildOutputFileFlag", flag.Type())
}

func TestBuildOutputSetStdin(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.SetStdin([]byte(`{"builds": [{"imageName": "gcr.io/k8s/test1", "tag": "sha256@foo"}]}`))

		flag := NewBuildOutputFileFlag("")
		err := flag.Set("-")
		t.CheckNoError(err)

		t.CheckDeepEqual(BuildOutput{
			Builds: []build.Artifact{{
				ImageName: "gcr.io/k8s/test1",
				Tag:       "sha256@foo",
			}},
		}, flag.buildOutput)
	})
}
