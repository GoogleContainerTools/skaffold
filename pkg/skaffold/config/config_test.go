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
	minimalConfig = `
apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: example
    workspace: ./examples/app
deploy:
  name: example
`
	minimalConfigWithTagger = `
apiVersion: skaffold/v1alpha2
kind: Config
build:
  tagPolicy:
    sha256: {}
  artifacts:
  - imageName: example
    workspace: ./examples/app
deploy:
  name: example
`
	badConfig = "bad config"
)

var (
	configShaTagger = &SkaffoldConfig{
		APIVersion: "skaffold/v1alpha2",
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
	configGitTagger = &SkaffoldConfig{
		APIVersion: "skaffold/v1alpha2",
		Kind:       "Config",
		Build: BuildConfig{
			TagPolicy: TagPolicy{GitTagger: &GitTagger{}},
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
)

func TestParseConfig(t *testing.T) {
	var tests = []struct {
		description string
		config      string
		dev         bool
		expected    *SkaffoldConfig
		badReader   bool
		shouldErr   bool
	}{
		{
			description: "Minimal config for dev",
			config:      minimalConfig,
			dev:         true,
			expected:    configShaTagger,
		},
		{
			description: "Minimal config for run",
			config:      minimalConfig,
			dev:         false,
			expected:    configGitTagger,
		},
		{
			description: "Minimal config with tagger for dev",
			config:      minimalConfigWithTagger,
			dev:         true,
			expected:    configShaTagger,
		},
		{
			description: "Minimal config with tagger for run",
			config:      minimalConfigWithTagger,
			dev:         false,
			expected:    configShaTagger,
		},
		{
			description: "Bad config",
			config:      badConfig,
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := Parse([]byte(test.config), test.dev)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, cfg)
		})
	}
}
