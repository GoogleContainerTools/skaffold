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

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestListBuilders(t *testing.T) {
	tests := []struct {
		description string
		build       *latestV2.BuildConfig
		expected    []string
	}{
		{
			description: "nil config",
			build:       nil,
			expected:    []string{},
		},
		{
			description: "multiple same builder config",
			build: &latestV2.BuildConfig{
				Artifacts: []*latestV2.Artifact{
					{ImageName: "img1", ArtifactType: latestV2.ArtifactType{DockerArtifact: &latestV2.DockerArtifact{}}},
					{ImageName: "img2", ArtifactType: latestV2.ArtifactType{DockerArtifact: &latestV2.DockerArtifact{}}},
				},
			},
			expected: []string{"docker"},
		},
		{
			description: "different builders config",
			build: &latestV2.BuildConfig{
				Artifacts: []*latestV2.Artifact{
					{ImageName: "img1", ArtifactType: latestV2.ArtifactType{DockerArtifact: &latestV2.DockerArtifact{}}},
					{ImageName: "img2", ArtifactType: latestV2.ArtifactType{JibArtifact: &latestV2.JibArtifact{}}},
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
		deploy      *latestV2.DeployConfig
		expected    []string
	}{
		{
			description: "nil config",
			deploy:      nil,
			expected:    []string{},
		},
		{
			description: "single deployer config",
			deploy: &latestV2.DeployConfig{
				DeployType: latestV2.DeployType{
					KubectlDeploy: &latestV2.KubectlDeploy{},
				},
			},
			expected: []string{"kubectl"},
		},
		{
			description: "multiple deployers config",
			deploy: &latestV2.DeployConfig{
				DeployType: latestV2.DeployType{
					LegacyHelmDeploy: &latestV2.LegacyHelmDeploy{},
					KubectlDeploy:    &latestV2.KubectlDeploy{},
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
