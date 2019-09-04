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

package schema

import (
	"fmt"
	"testing"

	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	yamlpatch "github.com/krishicks/yaml-patch"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestApplyPatch(t *testing.T) {
	config := `build:
  artifacts:
  - image: example
profiles:
- name: patches
  patches:
  - path: /build/artifacts/0/image
    value: replacement
  - op: add
    path: /build/artifacts/0/docker
    value:
      dockerfile: Dockerfile.DEV
  - op: add
    path: /build/artifacts/-
    value:
      image: second
      docker:
        dockerfile: Dockerfile.second
`

	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("skaffold.yaml", addVersion(config))

		parsed, err := ParseConfig(tmpDir.Path("skaffold.yaml"), false)
		t.CheckNoError(err)

		skaffoldConfig := parsed.(*latest.SkaffoldConfig)
		err = ApplyProfiles(skaffoldConfig, cfg.SkaffoldOptions{
			Profiles: []string{"patches"},
		})

		t.CheckNoError(err)
		t.CheckDeepEqual("replacement", skaffoldConfig.Build.Artifacts[0].ImageName)
		t.CheckDeepEqual("Dockerfile.DEV", skaffoldConfig.Build.Artifacts[0].DockerArtifact.DockerfilePath)
		t.CheckDeepEqual("Dockerfile.second", skaffoldConfig.Build.Artifacts[1].DockerArtifact.DockerfilePath)
	})
}

func TestApplyInvalidPatch(t *testing.T) {
	config := `build:
  artifacts:
  - image: example
profiles:
- name: patches
  patches:
  - path: /build/artifacts/0/image/
    value: replacement
`

	testutil.Run(t, "", func(t *testutil.T) {
		tmp := t.NewTempDir().
			Write("skaffold.yaml", addVersion(config))

		parsed, err := ParseConfig(tmp.Path("skaffold.yaml"), false)
		t.CheckNoError(err)

		skaffoldConfig := parsed.(*latest.SkaffoldConfig)
		err = ApplyProfiles(skaffoldConfig, cfg.SkaffoldOptions{
			Profiles: []string{"patches"},
		})

		t.CheckErrorAndDeepEqual(true, err, "applying profile patches: invalid path: /build/artifacts/0/image/", err.Error())
	})
}

func TestApplyProfiles(t *testing.T) {
	tests := []struct {
		description string
		config      *latest.SkaffoldConfig
		profile     string
		expected    *latest.SkaffoldConfig
		shouldErr   bool
	}{
		{
			description: "unknown profile",
			config:      config(),
			profile:     "profile",
			shouldErr:   true,
		},
		{
			description: "build type",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							BuildType: latest.BuildType{
								GoogleCloudBuild: &latest.GoogleCloudBuild{
									ProjectID:   "my-project",
									DockerImage: "gcr.io/cloud-builders/docker",
									MavenImage:  "gcr.io/cloud-builders/mvn",
									GradleImage: "gcr.io/cloud-builders/gradle",
								},
							},
						},
					},
				}),
			),
			expected: config(
				withGoogleCloudBuild("my-project",
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "tag policy",
			profile:     "dev",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "dev",
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
						},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withShaTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "artifacts",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "image", Workspace: ".", ArtifactType: latest.ArtifactType{
									DockerArtifact: &latest.DockerArtifact{
										DockerfilePath: "Dockerfile.DEV",
									},
								}},
								{ImageName: "imageProd", Workspace: ".", ArtifactType: latest.ArtifactType{
									DockerArtifact: &latest.DockerArtifact{
										DockerfilePath: "Dockerfile.DEV",
									},
								}},
							},
						},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile.DEV"),
					withDockerArtifact("imageProd", ".", "Dockerfile.DEV"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "deploy",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						Deploy: latest.DeployConfig{
							DeployType: latest.DeployType{
								HelmDeploy: &latest.HelmDeploy{},
							},
						},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withHelmDeploy(),
			),
		},
		{
			description: "patch Dockerfile",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Patches: []latest.JSONPatch{{
						Path:  "/build/artifacts/0/docker/dockerfile",
						Value: &util.YamlpatchNode{Node: *yamlpatch.NewNode(str("Dockerfile.DEV"))},
					}},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile.DEV"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "invalid patch path",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Patches: []latest.JSONPatch{{
						Path: "/unknown",
						Op:   "replace",
					}},
				}),
			),
			shouldErr: true,
		},
		{
			description: "add test case",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
				),
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						Test: []*latest.TestCase{{
							ImageName:      "image",
							StructureTests: []string{"test/*"},
						}},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withTests(&latest.TestCase{
					ImageName:      "image",
					StructureTests: []string{"test/*"},
				}),
			),
		},
		{
			description: "port forwarding",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
				),
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						PortForward: []*latest.PortForwardResource{{
							Namespace: "ns",
							Name:      "name",
							Type:      "service",
							Port:      8080,
							LocalPort: 8888,
						}},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withPortForward(&latest.PortForwardResource{
					Namespace: "ns",
					Name:      "name",
					Type:      "service",
					Port:      8080,
					LocalPort: 8888,
				}),
			),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := ApplyProfiles(test.config, cfg.SkaffoldOptions{
				Profiles: []string{test.profile},
			})

			if test.shouldErr {
				t.CheckError(test.shouldErr, err)
			} else {
				t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, test.config)
			}
		})
	}
}

func TestActivatedProfiles(t *testing.T) {
	tests := []struct {
		description string
		profiles    []latest.Profile
		opts        cfg.SkaffoldOptions
		expected    []string
		shouldErr   bool
	}{
		{
			description: "Selected on the command line",
			opts: cfg.SkaffoldOptions{
				Command:  "dev",
				Profiles: []string{"activated", "also-activated"},
			},
			profiles: []latest.Profile{
				{Name: "activated"},
				{Name: "not-activated"},
				{Name: "also-activated"},
			},
			expected: []string{"activated", "also-activated"},
		}, {
			description: "Auto-activated by command",
			opts: cfg.SkaffoldOptions{
				Command: "dev",
			},
			profiles: []latest.Profile{
				{Name: "run-profile", Activation: []latest.Activation{{Command: "run"}}},
				{Name: "dev-profile", Activation: []latest.Activation{{Command: "dev"}}},
				{Name: "non-run-profile", Activation: []latest.Activation{{Command: "!run"}}},
				{Name: "run-or-dev-profile", Activation: []latest.Activation{{Command: "(run)|(dev)"}}},
				{Name: "other-profile", Activation: []latest.Activation{{Command: "!(run)|(dev)"}}},
			},
			expected: []string{"dev-profile", "non-run-profile", "run-or-dev-profile"},
		}, {
			description: "Auto-activated by env variable",
			profiles: []latest.Profile{
				{Name: "activated", Activation: []latest.Activation{{Env: "KEY=VALUE"}}},
				{Name: "not-activated", Activation: []latest.Activation{{Env: "KEY=OTHER"}}},
				{Name: "also-activated", Activation: []latest.Activation{{Env: "KEY=!OTHER"}}},
				{Name: "regex-activated", Activation: []latest.Activation{{Env: "KEY=V.*E"}}},
			},
			expected: []string{"activated", "also-activated", "regex-activated"},
		}, {
			description: "Invalid env variable",
			opts:        cfg.SkaffoldOptions{},
			profiles: []latest.Profile{
				{Name: "activated", Activation: []latest.Activation{{Env: "KEY:VALUE"}}},
			},
			shouldErr: true,
		}, {
			description: "Auto-activated by kube context",
			profiles: []latest.Profile{
				{Name: "activated", Activation: []latest.Activation{{KubeContext: "prod-context"}}},
				{Name: "not-activated", Activation: []latest.Activation{{KubeContext: "dev-context"}}},
				{Name: "also-activated", Activation: []latest.Activation{{KubeContext: "!dev-context"}}},
				{Name: "activated-regexp", Activation: []latest.Activation{{KubeContext: "prod-.*"}}},
				{Name: "not-activated-regexp", Activation: []latest.Activation{{KubeContext: "dev-.*"}}},
				{Name: "invalid-regexp", Activation: []latest.Activation{{KubeContext: `\`}}},
			},
			expected: []string{"activated", "also-activated", "activated-regexp"},
		}, {
			description: "AND between activation criteria",
			opts: cfg.SkaffoldOptions{
				Command: "dev",
			},
			profiles: []latest.Profile{
				{
					Name: "activated", Activation: []latest.Activation{{
						Env:         "KEY=VALUE",
						KubeContext: "prod-context",
						Command:     "dev",
					}},
				},
				{
					Name: "not-activated", Activation: []latest.Activation{{
						Env:         "KEY=VALUE",
						KubeContext: "prod-context",
						Command:     "build",
					}},
				},
			},
			expected: []string{"activated"},
		}, {
			description: "OR between activations",
			opts: cfg.SkaffoldOptions{
				Command: "dev",
			},
			profiles: []latest.Profile{
				{
					Name: "activated", Activation: []latest.Activation{{
						Command: "run",
					}, {
						Command: "dev",
					}},
				},
			},
			expected: []string{"activated"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetEnvs(map[string]string{"KEY": "VALUE"})
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "prod-context"})

			activated, err := activatedProfiles(test.profiles, test.opts)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, activated)
		})
	}
}

func str(value string) *interface{} {
	var v interface{} = value
	return &v
}

func addVersion(yaml string) string {
	return fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latest.Version, yaml)
}
