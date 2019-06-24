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
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta7"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	invalidFileName     = "invalid-skaffold.yaml"
	validFileName       = "valid-skaffold.yaml"
	upgradeableFileName = "upgradeable-skaffold.yaml"
)

func TestFindConfigs(t *testing.T) {
	testutil.Run(t, "", func(tt *testutil.T) {
		latestVersion := latest.Version
		upgradeableVersion := v1beta7.Version
		tmpDir1, tmpDir2 := setUpTempFiles(tt, latestVersion, upgradeableVersion)

		tests := []struct {
			flagDir                *testutil.TempDir
			resultCounts           int
			shouldContainsMappings map[string]string
		}{
			{
				flagDir:                tmpDir1,
				resultCounts:           2,
				shouldContainsMappings: map[string]string{validFileName: latestVersion, upgradeableFileName: upgradeableVersion},
			},
			{
				flagDir:                tmpDir2,
				resultCounts:           1,
				shouldContainsMappings: map[string]string{validFileName: latestVersion},
			},
		}
		for _, test := range tests {
			pathToVersion, err := findConfigs(test.flagDir.Root())

			tt.CheckErrorAndDeepEqual(false, err, len(test.shouldContainsMappings), len(pathToVersion))
			for f, v := range test.shouldContainsMappings {
				version, ok := pathToVersion[test.flagDir.Path(f)]
				tt.CheckDeepEqual(true, ok)
				tt.CheckDeepEqual(version, v)
			}
		}
	})
}

/*
This helper function will generate the following file tree for testing purpose
...
├── tmpDir1
│   ├── valid-skaffold.yaml
|   ├── upgradeable-skaffold.yaml
│   └── invalid-skaffold.yaml
└── tmpDir2
	├── valid-skaffold.yaml
	└── invalid-skaffold.yaml
*/
func setUpTempFiles(t *testutil.T, latestVersion, upgradeableVersion string) (*testutil.TempDir, *testutil.TempDir) {
	validYaml := fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
`, latestVersion)

	upgradeableYaml := fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
`, upgradeableVersion)

	invalidYaml := `This is invalid`

	tmpDir1 := t.NewTempDir().WriteFiles(map[string]string{
		invalidFileName:     invalidYaml,
		validFileName:       validYaml,
		upgradeableFileName: upgradeableYaml,
	})

	tmpDir2 := t.NewTempDir().WriteFiles(map[string]string{
		invalidFileName: invalidYaml,
		validFileName:   validYaml,
	})

	return tmpDir1, tmpDir2
}
