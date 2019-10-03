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

package cluster

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		description        string
		artifact           *latest.KanikoArtifact
		insecureRegistries map[string]bool
		tag                string
		shouldErr          bool
		expectedArgs       []string
	}{
		{
			description: "simple build",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
			},
			expectedArgs: []string{},
		},
		{
			description: "cache layers",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache:          &latest.KanikoCache{},
			},
			expectedArgs: []string{"--cache=true"},
		},
		{
			description: "cache layers to specific repo",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache: &latest.KanikoCache{
					Repo: "repo",
				},
			},
			expectedArgs: []string{"--cache=true", "--cache-repo", "repo"},
		},
		{
			description: "cache path",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache: &latest.KanikoCache{
					HostPath: "/cache",
				},
			},
			expectedArgs: []string{"--cache=true", "--cache-dir", "/cache"},
		},
		{
			description: "target",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Target:         "target",
			},
			expectedArgs: []string{"--target", "target"},
		},
		{
			description: "reproducible",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Reproducible:   true,
			},
			expectedArgs: []string{"--reproducible"},
		},
		{
			description: "build args",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"nil_key":   nil,
					"empty_key": util.StringPtr(""),
					"value_key": util.StringPtr("value"),
				},
			},
			expectedArgs: []string{"--build-arg", "empty_key=", "--build-arg", "nil_key", "--build-arg", "value_key=value"},
		},
		{
			description: "invalid build args",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"invalid": util.StringPtr("{{Invalid"),
				},
			},
			shouldErr: true,
		},
		{
			description: "insecure registries",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
			},
			insecureRegistries: map[string]bool{"localhost:4000": true},
			expectedArgs:       []string{"--insecure-registry", "localhost:4000"},
		},
		{
			description: "skip tls",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SkipTLS:        true,
			},
			expectedArgs: []string{"--skip-tls-verify-registry", "gcr.io"},
		},
		{
			description: "invalid registry",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SkipTLS:        true,
			},
			tag:       "!!!!",
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			commonArgs := []string{"--dockerfile", "Dockerfile", "--context", "context", "--destination", "gcr.io/tag", "-v", "info"}

			tag := "gcr.io/tag"
			if test.tag != "" {
				tag = test.tag
			}
			args, err := args(test.artifact, "context", tag, test.insecureRegistries)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(append(commonArgs, test.expectedArgs...), args)
			}
		})
	}
}
