/*
Copyright 2018 Google LLC

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

package config

import (
	"io"
	"strings"
	"testing"

	testutil "github.com/GoogleCloudPlatform/skaffold/test"
)

const (
	rawConfigA = `
apiVersion: skaffold/v1
kind: Config
build:
  artifacts:
  - imageName: example
    workspace: ./examples/app
deploy:
  name: example
  parameters:
    key: value
`
	badConfigA = "bad config"
)

var configA = &SkaffoldConfig{
	APIVersion: "skaffold/v1",
	Kind:       "Config",
	Build: BuildConfig{
		Artifacts: []*Artifact{
			{
				ImageName: "example",
				Workspace: "./examples/app",
			},
		},
	},
	Deploy: DeployConfig{
		Name: "example",
		Parameters: map[string]string{
			"key": "value",
		},
	},
}

func TestParseConfig(t *testing.T) {
	var tests = []struct {
		description string
		config      string
		expected    *SkaffoldConfig
		badReader   bool
		shouldErr   bool
	}{
		{
			description: "Parse config",
			config:      rawConfigA,
			expected:    configA,
		},
		{
			description: "Bad config",
			config:      badConfigA,
			shouldErr:   true,
		},
		{
			description: "bad reader",
			badReader:   true,
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var r io.Reader
			r = strings.NewReader(test.config)
			if test.badReader {
				r = testutil.BadReader{}
			}
			cfg, err := Parse(r)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, cfg)
		})
	}
}
