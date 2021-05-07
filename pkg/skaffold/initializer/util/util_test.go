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

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestListBuilders(t *testing.T) {
	tests := []struct {
		description string
		build       *latestV1.BuildConfig
		expected    []string
	}{
		{
			description: "nil config",
			build:       nil,
			expected:    []string{},
		},
		{
			description: "multiple same builder config",
			build: &latestV1.BuildConfig{
				Artifacts: []*latestV1.Artifact{
					{ImageName: "img1", ArtifactType: latestV1.ArtifactType{DockerArtifact: &latestV1.DockerArtifact{}}},
					{ImageName: "img2", ArtifactType: latestV1.ArtifactType{DockerArtifact: &latestV1.DockerArtifact{}}},
				},
			},
			expected: []string{"docker"},
		},
		{
			description: "different builders config",
			build: &latestV1.BuildConfig{
				Artifacts: []*latestV1.Artifact{
					{ImageName: "img1", ArtifactType: latestV1.ArtifactType{DockerArtifact: &latestV1.DockerArtifact{}}},
					{ImageName: "img2", ArtifactType: latestV1.ArtifactType{JibArtifact: &latestV1.JibArtifact{}}},
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
		deploy      *latestV1.DeployConfig
		expected    []string
	}{
		{
			description: "nil config",
			deploy:      nil,
			expected:    []string{},
		},
		{
			description: "single deployer config",
			deploy: &latestV1.DeployConfig{
				DeployType: latestV1.DeployType{
					KubectlDeploy: &latestV1.KubectlDeploy{},
				},
			},
			expected: []string{"kubectl"},
		},
		{
			description: "multiple deployers config",
			deploy: &latestV1.DeployConfig{
				DeployType: latestV1.DeployType{
					HelmDeploy:    &latestV1.HelmDeploy{},
					KubectlDeploy: &latestV1.KubectlDeploy{},
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
