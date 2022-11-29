/*
Copyright 2022 The Skaffold Authors

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

package gcb

import (
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schema "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v2beta28"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestCloudBuildConfig(t *testing.T) {
	tests := []struct {
		description  string
		koImage      string
		skaffoldYaml string
		imageTag     string
		verbosity    string
		env          []string
	}{
		{
			description: "ensure Skaffold Config manifest is created in build step",
			koImage:     "gcr.io/k8s-skaffold/skaffold:v1.37.2-lts@sha256:0bde2b09928ce891f4e1bfb8d957648bbece9987ec6ef3678c6542196e64e71a",
			skaffoldYaml: `apiVersion: skaffold/v2beta28
kind: Config
build:
  artifacts:
  - image: skaffold-ko
    ko: {}
`,
			imageTag:  "mytag",
			verbosity: "info",
			env:       []string{"GOTRACEBACK=2"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			config := cloudBuildConfig(test.koImage, []byte(test.skaffoldYaml), test.imageTag, test.verbosity, test.env...)
			t.CheckEmpty(config.Images)
			step := config.Steps[0]
			t.CheckDeepEqual(test.koImage, step.Name)
			t.CheckDeepEqual("sh", step.Entrypoint)
			t.CheckContains(test.skaffoldYaml, step.Args[1])
			t.CheckContains("--tag "+test.imageTag, step.Args[1])
			t.CheckContains("--verbosity "+test.verbosity, step.Args[1])
			for _, envvar := range test.env {
				t.CheckContains(envvar, strings.Join(step.Env, " "))
			}
		})
	}
}

func TestCreateSkaffoldConfig(t *testing.T) {
	tests := []struct {
		description            string
		artifact               *latest.Artifact
		imageName              string
		insecureRegistries     []string
		platforms              []string
		expectedSkaffoldConfig *schema.SkaffoldConfig
	}{
		{
			description: "all fields",
			artifact: &latest.Artifact{
				ImageName: "myimage",
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						BaseImage: "baseImage",
						Dir:       "./dir",
						Env:       []string{"GOTRACEBACK=2"},
						Flags:     []string{"-tags", "netgo"},
						Labels: map[string]string{
							"org.opencontainers.image.source": "https://github.com/GoogleContainerTools/skaffold.git",
						},
						Ldflags: []string{"-s", "-w"},
						Main:    "./main",
						Dependencies: &latest.KoDependencies{
							Paths:  []string{"unused-koartifact-paths"},
							Ignore: []string{"unused-koartifact-ingore"},
						},
					},
				},
				Platforms: []string{"linux/amd64"},
				Workspace: "./workspace",
				Dependencies: []*latest.ArtifactDependency{{
					ImageName: "unused-dependency-image-name",
					Alias:     "unused-dependency-alias",
				}},
				LifecycleHooks: latest.BuildHooks{
					PreHooks: []latest.HostHook{{
						Command: []string{"unused-lifecycle-hook-command"},
					}},
				},
				Sync: &latest.Sync{
					Infer: []string{"unused-sync-infer"},
				},
			},
			imageName:          "gcr.io/project-id/myimage",
			insecureRegistries: []string{"insecure.example.com:5000"},
			platforms:          []string{"linux/amd64", "linux/arm64"},
			expectedSkaffoldConfig: &schema.SkaffoldConfig{
				APIVersion: schema.Version,
				Kind:       "Config",
				Pipeline: schema.Pipeline{
					Build: schema.BuildConfig{
						Artifacts: []*schema.Artifact{{
							ImageName: "gcr.io/project-id/myimage",
							ArtifactType: schema.ArtifactType{
								KoArtifact: &schema.KoArtifact{
									BaseImage: "baseImage",
									Dir:       "./dir",
									Env:       []string{"GOTRACEBACK=2"},
									Flags:     []string{"-tags", "netgo"},
									Labels: map[string]string{
										"org.opencontainers.image.source": "https://github.com/GoogleContainerTools/skaffold.git",
									},
									Ldflags: []string{"-s", "-w"},
									Main:    "./main",
								},
							},
							Platforms: []string{"linux/amd64"},
							Workspace: "./workspace",
						}},
						InsecureRegistries: []string{"insecure.example.com:5000"},
						Platforms:          []string{"linux/amd64", "linux/arm64"},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			skaffoldConfig := createSkaffoldConfig(test.artifact, test.imageName, test.platforms, test.insecureRegistries)
			t.CheckDeepEqual(test.expectedSkaffoldConfig, skaffoldConfig)
		})
	}
}

func TestSplitImageNameAndTag(t *testing.T) {
	tests := []struct {
		description  string
		input        string
		expectedName string
		expectedTag  string
	}{
		{
			description:  "image name with tag",
			input:        "gcr.io/project-id/myimage:mytag",
			expectedName: "gcr.io/project-id/myimage",
			expectedTag:  "mytag",
		},
		{
			description:  "image name without tag defaults to latest",
			input:        "gcr.io/project-id/myimage",
			expectedName: "gcr.io/project-id/myimage",
			expectedTag:  "latest",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			name, tag := splitImageNameAndTag(test.input)
			t.CheckDeepEqual(test.expectedName, name)
			t.CheckDeepEqual(test.expectedTag, tag)
		})
	}
}
