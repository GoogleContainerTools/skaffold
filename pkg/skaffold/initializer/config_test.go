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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type stubDeploymentInitializer struct {
	config   latest.DeployConfig
	profiles []latest.Profile
}

func (s stubDeploymentInitializer) DeployConfig() (latest.DeployConfig, []latest.Profile) {
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
	pairs []build.BuilderImagePair
}

func (s stubBuildInitializer) ProcessImages([]string) error {
	panic("no")
}

func (s stubBuildInitializer) PrintAnalysis(io.Writer) error {
	panic("no sir")
}

func (s stubBuildInitializer) BuildConfig() latest.BuildConfig {
	return latest.BuildConfig{
		Artifacts: build.Artifacts(s.pairs),
	}
}

func (s stubBuildInitializer) GenerateManifests() (map[build.GeneratedBuilderImagePair][]byte, error) {
	panic("no thank you")
}

func TestGenerateSkaffoldConfig(t *testing.T) {
	tests := []struct {
		name                   string
		expectedSkaffoldConfig *latest.SkaffoldConfig
		deployConfig           latest.DeployConfig
		profiles               []latest.Profile
		builderConfigPairs     []build.BuilderImagePair
		getWd                  func() (string, error)
	}{
		{
			name:               "empty",
			builderConfigPairs: []build.BuilderImagePair{},
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
			builderConfigPairs: []build.BuilderImagePair{
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
			builderConfigPairs: []build.BuilderImagePair{},
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
				test.profiles,
			}
			buildInitializer := stubBuildInitializer{
				test.builderConfigPairs,
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
