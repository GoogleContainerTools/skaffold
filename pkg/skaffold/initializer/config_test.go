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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type stubDeploymentInitializer struct {
	config latest.DeployConfig
}

func (s stubDeploymentInitializer) DeployConfig() latest.DeployConfig {
	return s.config
}

type stubRendererInitializer struct {
	config   latest.RenderConfig
	profiles []latest.Profile
}

func (s stubRendererInitializer) RenderConfig() (latest.RenderConfig, []latest.Profile) {
	return s.config, s.profiles
}
func (s stubRendererInitializer) GetImages() []string {
	panic("implement me")
}

func (s stubRendererInitializer) Validate() error {
	panic("no thanks")
}

func (s stubRendererInitializer) AddManifestForImage(string, string) {
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

func (s stubBuildInitializer) BuildConfig() (latest.BuildConfig, []*latest.PortForwardResource) {
	return latest.BuildConfig{
		Artifacts: build.Artifacts(s.artifactInfos),
	}, nil
}

func (s stubBuildInitializer) GenerateManifests(io.Writer, bool, bool) (map[build.GeneratedArtifactInfo][]byte, error) {
	panic("no thank you")
}

func TestGenerateSkaffoldConfig(t *testing.T) {
	tests := []struct {
		name                   string
		expectedSkaffoldConfig *latest.SkaffoldConfig
		deployConfig           latest.DeployConfig
		renderConfig           latest.RenderConfig
		profiles               []latest.Profile
		builderConfigInfos     []build.ArtifactInfo
		getWd                  func() (string, error)
	}{
		{
			name:               "empty",
			builderConfigInfos: []build.ArtifactInfo{},
			deployConfig:       latest.DeployConfig{},
			renderConfig:       latest.RenderConfig{},
			getWd: func() (s string, err error) {
				return filepath.Join("rootDir", "testConfig"), nil
			},
			expectedSkaffoldConfig: &latest.SkaffoldConfig{
				APIVersion: latest.Version,
				Kind:       "Config",
				Metadata:   latest.Metadata{Name: "testconfig"},
				Pipeline: latest.Pipeline{
					Render: latest.RenderConfig{},
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
			deployConfig: latest.DeployConfig{},
			renderConfig: latest.RenderConfig{},
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
								ArtifactType: latest.ArtifactType{
									DockerArtifact: &latest.DockerArtifact{DockerfilePath: "Dockerfile"},
								},
							},
						},
					},
					Render: latest.RenderConfig{},
					Deploy: latest.DeployConfig{},
				},
			},
		},
		{
			name:               "error working dir",
			builderConfigInfos: []build.ArtifactInfo{},
			deployConfig:       latest.DeployConfig{},
			renderConfig:       latest.RenderConfig{},
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
			buildInitializer := stubBuildInitializer{
				test.builderConfigInfos,
			}
			rendererInitializer := stubRendererInitializer{
				test.renderConfig,
				test.profiles,
			}
			t.Override(&getWd, test.getWd)
			config := generateSkaffoldConfig(buildInitializer, rendererInitializer, deploymentInitializer)
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
