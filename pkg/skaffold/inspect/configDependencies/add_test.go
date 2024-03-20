/*
Copyright 2023 The Skaffold Authors

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

package inspect

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	// Base config that can be used by config dependencies referenced.
	baseCfg = `apiVersion: skaffold/v4beta5
kind: Config`
	// We use the latest API version in the tests because the parser upgrades the version.
	apiVersion = latest.Version
)

func TestAddConfigDependencies(t *testing.T) {
	tests := []struct {
		description            string
		config                 string
		existingConfigDepFiles []string
		input                  string
		modules                []string
		expected               string
		shouldErr              bool
	}{
		{
			description: "adds remote config dependency",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
`, apiVersion),
			input: `
{
  "dependencies": [
    {
	  "configs": ["c1"],
	  "path": "add-dep.yaml"
	}
  ]
}`,
			expected: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
requires:
  - configs:
      - c1
    path: add-dep.yaml
`, apiVersion),
		},
		{
			description: "adds remote config dependencies when requires is present",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg2
requires:
  - path: existing-dep.yaml
`, apiVersion),
			existingConfigDepFiles: []string{"existing-dep.yaml"},
			input: `
{
  "dependencies": [
	{
	  "configs": ["c1"],
	  "path": "/add-dep.yaml"
	},
	{
	  "path": "/add-dep-2.yaml"
	}
  ]
}`,
			expected: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg2
requires:
  - path: existing-dep.yaml
  - configs:
      - c1
    path: /add-dep.yaml
  - path: /add-dep-2.yaml
`, apiVersion),
		},
		{
			description: "adds remote config dependency to multiple configs when module unspecified",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
---
apiVersion: %s
kind: Config
metadata:
  name: cfg1_1
`, apiVersion, apiVersion),
			input: `
{
  "dependencies": [
    {
	  "configs": ["c1"],
	  "path": "/add-dep.yaml"
	}
  ]
}`,
			expected: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
requires:
  - configs:
      - c1
    path: /add-dep.yaml
---
apiVersion: %s
kind: Config
metadata:
  name: cfg1_1
requires:
  - configs:
      - c1
    path: /add-dep.yaml
`, apiVersion, apiVersion),
		},
		{
			description: "adds remote config dependency only to specified module",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
---
apiVersion: %s
kind: Config
metadata:
  name: cfg1_1
`, apiVersion, apiVersion),
			input: `
{
  "dependencies": [
    {
	  "configs": ["c1"],
	  "path": "/add-dep.yaml"
	}
  ]
}`,
			modules: []string{"cfg1"},
			expected: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
requires:
  - configs:
      - c1
    path: /add-dep.yaml
---
apiVersion: %s
kind: Config
metadata:
  name: cfg1_1
`, apiVersion, apiVersion),
		},
		{
			description: "adds GoogleCloudBuildRepoV2 remote config dependency",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
---
apiVersion: %s
kind: Config
metadata:
  name: cfg2
`, apiVersion, apiVersion),
			input: `
{
  "dependencies": [
    {
	  "configs": ["c1"],
      "googleCloudBuildRepoV2": {
        "projectID": "k8s-skaffold",
        "region": "us-central1",
        "connection": "github-connection-e2e-tests",
        "repo": "skaffold-getting-started",
        "path": "skaffold.yaml",
        "ref": "main"
	  }
	}
  ]
}`,
			modules: []string{"cfg1"},
			expected: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
  name: cfg1
requires:
  - configs:
      - c1
    googleCloudBuildRepoV2:
      projectID: k8s-skaffold
      region: us-central1
      connection: github-connection-e2e-tests
      repo: skaffold-getting-started
      path: skaffold.yaml
      ref: main
      sync: false
---
apiVersion: %s
kind: Config
metadata:
  name: cfg2
`, apiVersion, apiVersion),
		},
		{
			description: "fails when specified module not present",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
metadata:
    name: cfg1
`, apiVersion),
			input: `
{
  "dependencies": [
    {
      "configs": ["c1"],
	  "path": "add-dep.yaml"
	}
  ]
}`,
			modules:   []string{"no"},
			shouldErr: true,
		},
		{
			description: "fails when input file is not list of config dependencies",
			config: fmt.Sprintf(`apiVersion: %s
kind: Config
`, apiVersion),
			input:     `input`,
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			td := t.NewTempDir()
			td.Write("skaffold.yaml", test.config)
			td.Write("input.yaml", test.input)
			configFile := td.Root() + "/skaffold.yaml"
			inputFile := td.Root() + "/input.yaml"

			// The existing dependency files need to exist for the parser to resolve the config.
			for _, s := range test.existingConfigDepFiles {
				td.Write(s, baseCfg)
			}

			var b bytes.Buffer
			err := AddConfigDependencies(context.Background(), &b, inspect.Options{Filename: configFile, Modules: test.modules}, inputFile)
			t.CheckError(test.shouldErr, err)
			if err == nil {
				// The original config file is updated so check it's as expected.
				t.CheckFileExistAndContent(configFile, []byte(test.expected))
			}
		})
	}
}
