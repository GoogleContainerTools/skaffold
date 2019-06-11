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

package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta7"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	invalidFileName    string = "invalid-skaffold.yaml"
	validFileName      string = "valid-skaffold.yaml"
	upgradableFileName string = "upgradable-skaffold.yaml"
)

func TestFindConfigs(t *testing.T) {
	testutil.Run(t, "", func(tt *testutil.T) {
		latestVersion := latest.Version
		upgradableVersion := v1beta7.Version
		tmpDir1, tmpDir2 := setUpTempFiles(tt, latestVersion, upgradableVersion)

		tests := []struct {
			flagDir                *testutil.TempDir
			resultCounts           int
			shouldContainsFiles    []string
			shouldContainsVersions []string
		}{
			{
				flagDir:                tmpDir1,
				resultCounts:           2,
				shouldContainsFiles:    []string{validFileName, upgradableFileName},
				shouldContainsVersions: []string{latestVersion, upgradableVersion},
			},
			{
				flagDir:                tmpDir2,
				resultCounts:           1,
				shouldContainsFiles:    []string{validFileName},
				shouldContainsVersions: []string{latestVersion},
			},
		}
		for _, test := range tests {
			var b bytes.Buffer
			err := findConfigs(&b, test.flagDir.Root())

			for _, f := range test.shouldContainsFiles {
				tt.CheckContains(test.flagDir.Path(f), b.String())
			}

			for _, v := range test.shouldContainsVersions {
				tt.CheckContains(v, b.String())
			}

			tt.CheckDeepEqual(test.resultCounts, strings.Count(b.String(), "\n"))
			tt.CheckError(false, err)
		}
	})
}

/*
This helper function will generate the following file tree for testing purpose
...
├── tmpDir1
│   ├── valid-skaffold.yaml
|   ├── upgradable-skaffold.yaml
│   └── invalid-skaffold.yaml
└── tmpDir2
	├── valid-skaffold.yaml
	└── invalid-skaffold.yaml
*/
func setUpTempFiles(tt *testutil.T, latestVersion, upgradableVersion string) (*testutil.TempDir, *testutil.TempDir) {
	validYaml := fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
`, latestVersion)
	upgradableYaml := fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
`, upgradableVersion)
	invalidYaml := `This is invalid`

	tmpDir1 := tt.NewTempDir()
	tmpDir2 := tt.NewTempDir()

	files := []struct {
		fileName string
		content  string
		tmpDir   *testutil.TempDir
	}{
		{
			fileName: invalidFileName,
			content:  invalidYaml,
			tmpDir:   tmpDir1,
		},
		{
			fileName: validFileName,
			content:  validYaml,
			tmpDir:   tmpDir1,
		},
		{
			fileName: upgradableFileName,
			content:  upgradableYaml,
			tmpDir:   tmpDir1,
		},
		{
			fileName: invalidFileName,
			content:  invalidYaml,
			tmpDir:   tmpDir2,
		},
		{
			fileName: validFileName,
			content:  validYaml,
			tmpDir:   tmpDir2,
		},
	}

	for _, file := range files {
		file.tmpDir.Write(file.fileName, file.content)
	}

	return tmpDir1, tmpDir2
}
