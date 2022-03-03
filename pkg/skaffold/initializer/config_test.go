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
	"io"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type stubDeploymentInitializer struct {
	config   latestV1.DeployConfig
	profiles []latestV1.Profile
}

func (s stubDeploymentInitializer) DeployConfig() (latestV1.DeployConfig, []latestV1.Profile) {
	return s.config, s.profiles
}

func (s stubDeploymentInitializer) GetImages() []string {
	panic("implement me")
}

func (s stubDeploymentInitializer) Validate() error {
	panic("no thanks")
}

func (s stubDeploymentInitializer) AddManifestForImage(string, string) {
	panic("don't call me")
}

type stubBuildInitializer struct {
	artifactInfos []build.ArtifactInfo
}

func (s stubBuildInitializer) ProcessImages([]string) error {
	panic("no")
}

func (s stubBuildInitializer) PrintAnalysis(io.Writer) error {
	panic("no sir")
}

func (s stubBuildInitializer) BuildConfig() (latestV1.BuildConfig, []*latestV1.PortForwardResource) {
	return latestV1.BuildConfig{
		Artifacts: build.Artifacts(s.artifactInfos),
	}, nil
}

func (s stubBuildInitializer) GenerateManifests(io.Writer, bool, bool) (map[build.GeneratedArtifactInfo][]byte, error) {
	panic("no thank you")
}

func TestGenerateSkaffoldConfig(t *testing.T) {
	tests := []struct {
		name                   string
		expectedSkaffoldConfig *latestV1.SkaffoldConfig
		deployConfig           latestV1.DeployConfig
		profiles               []latestV1.Profile
		builderConfigInfos     []build.ArtifactInfo
		getWd                  func() (string, error)
	}{
		{
			name:               "empty",
			builderConfigInfos: []build.ArtifactInfo{},
			deployConfig:       latestV1.DeployConfig{},
			getWd: func() (s string, err error) {
				return filepath.Join("rootDir", "testConfig"), nil
			},
			expectedSkaffoldConfig: &latestV1.SkaffoldConfig{
				APIVersion: latestV1.Version,
				Kind:       "Config",
				Metadata:   latestV1.Metadata{Name: "testconfig"},
				Pipeline: latestV1.Pipeline{
					Deploy: latestV1.DeployConfig{},
				},
			},
		},
		{
			name: "root dir + builder image pairs",
			builderConfigInfos: []build.ArtifactInfo{
				{
					Builder: docker.ArtifactConfig{
						File: "testDir/Dockerfile",
					},
					ImageName: "image1",
				},
			},
			deployConfig: latestV1.DeployConfig{},
			getWd: func() (s string, err error) {
				return string(filepath.Separator), nil
			},
			expectedSkaffoldConfig: &latestV1.SkaffoldConfig{
				APIVersion: latestV1.Version,
				Kind:       "Config",
				Metadata:   latestV1.Metadata{},
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "image1",
								Workspace: "testDir",
								ArtifactType: latestV1.ArtifactType{
									DockerArtifact: &latestV1.DockerArtifact{DockerfilePath: "Dockerfile"},
								},
							},
						},
					},
					Deploy: latestV1.DeployConfig{},
				},
			},
		},
		{
			name:               "error working dir",
			builderConfigInfos: []build.ArtifactInfo{},
			deployConfig:       latestV1.DeployConfig{},
			getWd: func() (s string, err error) {
				return "", errors.New("testError")
			},
			expectedSkaffoldConfig: &latestV1.SkaffoldConfig{
				APIVersion: latestV1.Version,
				Kind:       "Config",
				Metadata:   latestV1.Metadata{},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			deploymentInitializer := stubDeploymentInitializer{
				test.deployConfig,
				test.profiles,
			}
			buildInitializer := stubBuildInitializer{
				test.builderConfigInfos,
			}
			t.Override(&getWd, test.getWd)
			config := generateSkaffoldConfig(buildInitializer, deploymentInitializer)
			t.CheckDeepEqual(config, test.expectedSkaffoldConfig)
		})
	}
}

func Test_canonicalizeName(t *testing.T) {
	const length253 = "aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa-aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa-aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaaaaaaaa.aaa"
	tests := []struct {
		in, out string
	}{
		{
			in:  "abc def",
			out: "abc-def",
		},
		{
			in:  "abc    def",
			out: "abc-def",
		},
		{
			in:  "abc...def",
			out: "abc...def",
		},
		{
			in:  "abc---def",
			out: "abc---def",
		},
		{
			in:  "aBc DeF",
			out: "abc-def",
		},
		{
			in:  length253 + "XXXXXXX",
			out: length253,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.in, func(t *testutil.T) {
			actual := canonicalizeName(test.in)

			t.CheckDeepEqual(test.out, actual)
		})
	}
}
