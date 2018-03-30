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
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

const (
	rawConfigA = `
apiVersion: skaffold/v1alpha1
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
	APIVersion: "skaffold/v1alpha1",
	Kind:       "Config",
	Build: BuildConfig{
		TagPolicy: TagPolicy{ShaTagger: &ShaTagger{}},
		Artifacts: []*Artifact{
			{
				ImageName: "example",
				Workspace: "./examples/app",
			},
		},
	},
	Deploy: DeployConfig{
		Name: "example",
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
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := Parse([]byte(test.config), DefaultDevSkaffoldConfig)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, cfg)
		})
	}
}
