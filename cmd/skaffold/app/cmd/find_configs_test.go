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
	"context"
	"fmt"
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta7"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFindConfigs(t *testing.T) {
	tests := []struct {
		files    map[string]string
		expected map[string]string
	}{
		{
			files: map[string]string{
				"valid.yml":        validYaml(latestV1.Version),
				"upgradeable.yaml": validYaml(v1beta7.Version),
				"invalid.yaml":     invalidYaml(),
			},
			expected: map[string]string{
				"valid.yml":        latestV1.Version,
				"upgradeable.yaml": v1beta7.Version,
			},
		},
		{
			files: map[string]string{
				"valid.yaml":   validYaml(latestV1.Version),
				"invalid.yaml": invalidYaml(),
			},
			expected: map[string]string{
				"valid.yaml": latestV1.Version,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			tmpDir := t.NewTempDir().WriteFiles(test.files)

			pathToVersion, err := findConfigs(context.TODO(), tmpDir.Root())

			t.CheckNoError(err)
			t.CheckDeepEqual(len(test.expected), len(pathToVersion))

			for f, v := range test.expected {
				version := pathToVersion[tmpDir.Path(f)]

				t.CheckDeepEqual(version, v)
			}
		})
	}
}

func validYaml(version string) string {
	return fmt.Sprintf("apiVersion: %s\nkind: Config", version)
}

func invalidYaml() string {
	return "This is invalid"
}
