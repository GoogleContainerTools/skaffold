/*
Copyright 2020 The Skaffold Authors

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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestListBuilders(t *testing.T) {
	tests := []struct {
		description string
		build       *latest.BuildConfig
		expected    []string
	}{
		{
			description: "nil config",
			build:       nil,
			expected:    []string{},
		},
		{
			description: "multiple same builder config",
			build: &latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{ImageName: "img1", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
					{ImageName: "img2", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
				},
			},
			expected: []string{"docker"},
		},
		{
			description: "different builders config",
			build: &latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{ImageName: "img1", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
					{ImageName: "img2", ArtifactType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{}}},
				},
			},
			expected: []string{"docker", "jib"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got := ListBuilders(test.build)
			t.CheckDeepEqual(test.expected, got)
		})
	}
}

func TestListDeployers(t *testing.T) {
	tests := []struct {
		description string
		deploy      *latest.DeployConfig
		expected    []string
	}{
		{
			description: "nil config",
			deploy:      nil,
			expected:    []string{},
		},
		{
			description: "single deployer config",
			deploy: &latest.DeployConfig{
				DeployType: latest.DeployType{
					KubectlDeploy: &latest.KubectlDeploy{},
				},
			},
			expected: []string{"kubectl"},
		},
		{
			description: "multiple deployers config",
			deploy: &latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy:    &latest.HelmDeploy{},
					KubectlDeploy: &latest.KubectlDeploy{},
				},
			},
			expected: []string{"helm", "kubectl"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got := ListDeployers(test.deploy)
			t.CheckDeepEqual(test.expected, got)
		})
	}
}
