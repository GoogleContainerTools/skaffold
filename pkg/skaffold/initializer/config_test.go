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
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type stubDeploymentInitializer struct {
	config   latest_v1.DeployConfig
	profiles []latest_v1.Profile
}

func (s stubDeploymentInitializer) DeployConfig() (latest_v1.DeployConfig, []latest_v1.Profile) {
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

func (s stubBuildInitializer) BuildConfig() (latest_v1.BuildConfig, []*latest_v1.PortForwardResource) {
	return latest_v1.BuildConfig{
		Artifacts: build.Artifacts(s.artifactInfos),
	}, nil
}

func (s stubBuildInitializer) GenerateManifests(io.Writer, bool) (map[build.GeneratedArtifactInfo][]byte, error) {
	panic("no thank you")
}

func TestGenerateSkaffoldConfig(t *testing.T) {
	tests := []struct {
		name                   string
		expectedSkaffoldConfig *latest_v1.SkaffoldConfig
		deployConfig           latest_v1.DeployConfig
		profiles               []latest_v1.Profile
		builderConfigInfos     []build.ArtifactInfo
		getWd                  func() (string, error)
	}{
		{
			name:               "empty",
			builderConfigInfos: []build.ArtifactInfo{},
			deployConfig:       latest_v1.DeployConfig{},
			getWd: func() (s string, err error) {
				return filepath.Join("rootDir", "testConfig"), nil
			},
			expectedSkaffoldConfig: &latest_v1.SkaffoldConfig{
				APIVersion: latest_v1.Version,
				Kind:       "Config",
				Metadata:   latest_v1.Metadata{Name: "testconfig"},
				Pipeline: latest_v1.Pipeline{
					Deploy: latest_v1.DeployConfig{},
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
			deployConfig: latest_v1.DeployConfig{},
			getWd: func() (s string, err error) {
				return string(filepath.Separator), nil
			},
			expectedSkaffoldConfig: &latest_v1.SkaffoldConfig{
				APIVersion: latest_v1.Version,
				Kind:       "Config",
				Metadata:   latest_v1.Metadata{},
				Pipeline: latest_v1.Pipeline{
					Build: latest_v1.BuildConfig{
						Artifacts: []*latest_v1.Artifact{
							{
								ImageName: "image1",
								Workspace: "testDir",
								ArtifactType: latest_v1.ArtifactType{
									DockerArtifact: &latest_v1.DockerArtifact{DockerfilePath: "Dockerfile"},
								},
							},
						},
					},
					Deploy: latest_v1.DeployConfig{},
				},
			},
		},
		{
			name:               "error working dir",
			builderConfigInfos: []build.ArtifactInfo{},
			deployConfig:       latest_v1.DeployConfig{},
			getWd: func() (s string, err error) {
				return "", errors.New("testError")
			},
			expectedSkaffoldConfig: &latest_v1.SkaffoldConfig{
				APIVersion: latest_v1.Version,
				Kind:       "Config",
				Metadata:   latest_v1.Metadata{},
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
