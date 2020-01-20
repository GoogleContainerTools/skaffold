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

package initializer

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type stubDeploymentInitializer struct {
	deployConfig latest.DeployConfig
}

func (s stubDeploymentInitializer) GenerateDeployConfig() latest.DeployConfig {
	return s.deployConfig
}

func (s stubDeploymentInitializer) GetImages() []string {
	panic("implement me")
}

func TestGenerateSkaffoldConfig(t *testing.T) {
	tests := []struct {
		name                   string
		expectedSkaffoldConfig *latest.SkaffoldConfig
		deployConfig           latest.DeployConfig
		builderConfigPairs     []builderImagePair
		getWd                  func() (string, error)
	}{
		{
			name:               "empty",
			builderConfigPairs: []builderImagePair{},
			deployConfig:       latest.DeployConfig{},
			getWd: func() (s string, err error) {
				return filepath.Join("rootDir", "testConfig"), nil
			},
			expectedSkaffoldConfig: &latest.SkaffoldConfig{
				APIVersion: latest.Version,
				Kind:       "Config",
				Metadata:   latest.Metadata{Name: "testconfig"},
				Pipeline: latest.Pipeline{
					Deploy: latest.DeployConfig{},
				},
			},
		},
		{
			name: "root dir + builder image pairs",
			builderConfigPairs: []builderImagePair{
				{
					Builder: docker.ArtifactConfig{
						File: "testDir/Dockerfile",
					},
					ImageName: "image1",
				},
			},
			deployConfig: latest.DeployConfig{},
			getWd: func() (s string, err error) {
				return string(filepath.Separator), nil
			},
			expectedSkaffoldConfig: &latest.SkaffoldConfig{
				APIVersion: latest.Version,
				Kind:       "Config",
				Metadata:   latest.Metadata{},
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "image1",
								Workspace: "testDir",
							},
						},
					},
					Deploy: latest.DeployConfig{},
				},
			},
		},
		{
			name:               "error working dir",
			builderConfigPairs: []builderImagePair{},
			deployConfig:       latest.DeployConfig{},
			getWd: func() (s string, err error) {
				return "", errors.New("testError")
			},
			expectedSkaffoldConfig: &latest.SkaffoldConfig{
				APIVersion: latest.Version,
				Kind:       "Config",
				Metadata:   latest.Metadata{},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			deploymentInitializer := stubDeploymentInitializer{
				test.deployConfig,
			}
			t.Override(&getWd, test.getWd)
			config := generateSkaffoldConfig(deploymentInitializer, test.builderConfigPairs)
			t.CheckDeepEqual(config, test.expectedSkaffoldConfig)
		})
	}
}

func TestArtifacts(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		artifacts := artifacts([]builderImagePair{
			{
				ImageName: "image1",
				Builder: docker.ArtifactConfig{
					File: "Dockerfile",
				},
			},
			{
				ImageName: "image2",
				Builder: docker.ArtifactConfig{
					File: "front/Dockerfile2",
				},
			},
			{
				ImageName: "image3",
				Builder: buildpacks.ArtifactConfig{
					File: "package.json",
				},
			},
		})

		expected := []*latest.Artifact{
			{
				ImageName:    "image1",
				ArtifactType: latest.ArtifactType{},
			},
			{
				ImageName: "image2",
				Workspace: "front",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile2",
					},
				},
			},
			{
				ImageName: "image3",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{
						Builder: "heroku/buildpacks",
					},
				},
			},
		}

		t.CheckDeepEqual(expected, artifacts)
	})
}
