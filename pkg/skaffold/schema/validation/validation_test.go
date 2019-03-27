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

package validation

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	cfgWithErrors = &latest.SkaffoldPipeline{
		Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{
				{
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{},
						BazelArtifact:  &latest.BazelArtifact{},
					},
				},
				{
					ArtifactType: latest.ArtifactType{
						BazelArtifact:  &latest.BazelArtifact{},
						KanikoArtifact: &latest.KanikoArtifact{},
					},
				},
			},
		},
		Deploy: latest.DeployConfig{
			DeployType: latest.DeployType{
				HelmDeploy:    &latest.HelmDeploy{},
				KubectlDeploy: &latest.KubectlDeploy{},
			},
		},
	}
)

func TestValidateSchema(t *testing.T) {
	err := ValidateSchema(cfgWithErrors)
	testutil.CheckError(t, true, err)

	err = ValidateSchema(&latest.SkaffoldPipeline{})
	testutil.CheckError(t, false, err)
}

func TestValidateOneOf(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		shouldErr bool
		expected  []string
	}{
		{
			name: "only one field set",
			input: &latest.BuildType{
				Cluster: &latest.ClusterDetails{},
			},
			shouldErr: false,
		},
		{
			name: "two colliding buildTypes",
			input: &latest.BuildType{
				GoogleCloudBuild: &latest.GoogleCloudBuild{},
				Cluster:          &latest.ClusterDetails{},
			},
			shouldErr: true,
			expected:  []string{"googleCloudBuild cluster"},
		},
		{
			name: "deployType with two fields",
			input: &latest.DeployType{
				HelmDeploy:    &latest.HelmDeploy{},
				KubectlDeploy: &latest.KubectlDeploy{},
			},
			shouldErr: true,
			expected:  []string{"helm kubectl"},
		},
		{
			name: "deployType with three fields",
			input: &latest.DeployType{
				HelmDeploy:      &latest.HelmDeploy{},
				KubectlDeploy:   &latest.KubectlDeploy{},
				KustomizeDeploy: &latest.KustomizeDeploy{},
			},
			shouldErr: true,
			expected:  []string{"helm kubectl kustomize"},
		},
		{
			name:      "empty struct should not fail",
			input:     &latest.GitTagger{},
			shouldErr: false,
		},
		{
			name:      "full Skaffold pipeline",
			input:     cfgWithErrors,
			shouldErr: true,
			expected:  []string{"docker bazel", "bazel kaniko", "helm kubectl"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := validateOneOf(test.input)

			if test.shouldErr {
				testutil.CheckDeepEqual(t, len(test.expected), len(actual))
				for i, message := range test.expected {
					testutil.CheckContains(t, message, actual[i].Error())
				}
			} else {
				testutil.CheckDeepEqual(t, []error(nil), actual)
			}
		})
	}
}
