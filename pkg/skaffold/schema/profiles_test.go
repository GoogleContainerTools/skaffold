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

	yamlpatch "github.com/krishicks/yaml-patch"
	"k8s.io/client-go/tools/clientcmd/api"

	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestApplyPatch(t *testing.T) {
	config := `build:
  artifacts:
  - image: example
deploy:
  kubectl:
    manifests:
    - k8s-*
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
  - op: remove
    path: /deploy
`

	testutil.Run(t, "", func(t *testutil.T) {
		setupFakeKubeConfig(t, api.Config{CurrentContext: "prod-context"})
		tmpDir := t.NewTempDir().
			Write("skaffold.yaml", addVersion(config))

		parsed, err := ParseConfig(tmpDir.Path("skaffold.yaml"))
		t.CheckNoError(err)

		skaffoldConfig := parsed.(*latest.SkaffoldConfig)
		err = ApplyProfiles(skaffoldConfig, cfg.SkaffoldOptions{
			Profiles: []string{"patches"},
		})

		t.CheckNoError(err)
		t.CheckDeepEqual("replacement", skaffoldConfig.Build.Artifacts[0].ImageName)
		t.CheckDeepEqual("Dockerfile.DEV", skaffoldConfig.Build.Artifacts[0].DockerArtifact.DockerfilePath)
		t.CheckDeepEqual("Dockerfile.second", skaffoldConfig.Build.Artifacts[1].DockerArtifact.DockerfilePath)
		t.CheckDeepEqual(latest.DeployConfig{}, skaffoldConfig.Deploy)
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

		parsed, err := ParseConfig(tmp.Path("skaffold.yaml"))
		t.CheckNoError(err)

		skaffoldConfig := parsed.(*latest.SkaffoldConfig)
		err = ApplyProfiles(skaffoldConfig, cfg.SkaffoldOptions{
			Profiles: []string{"patches"},
		})

		t.CheckErrorAndDeepEqual(true, err, `applying profile "patches": invalid path: /build/artifacts/0/image/`, err.Error())
	})
}

func TestApplyProfiles(t *testing.T) {
	tests := []struct {
		description              string
		config                   *latest.SkaffoldConfig
		profile                  string
		expected                 *latest.SkaffoldConfig
		kubeContextCli           string
		profileAutoActivationCli bool
		shouldErr                bool
	}{
		{
			description: "unknown profile",
			config:      config(),
			profile:     "profile",
			shouldErr:   true,
		},
		{
			description:              "build type",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
									KanikoImage: constants.DefaultKanikoImage,
									PackImage:   "gcr.io/k8s-skaffold/pack",
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
			description:              "tag policy",
			profile:                  "dev",
			profileAutoActivationCli: true,
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
			description:              "artifacts",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
			description:              "deploy",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description:              "patch Dockerfile",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
			description:              "invalid patch path",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
			description:              "add test case",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
			description:              "port forwarding",
			profile:                  "profile",
			profileAutoActivationCli: true,
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
		{
			description:              "activate kubecontext specific profile and change the kubecontext",
			profile:                  "profile",
			profileAutoActivationCli: true,
			config: config(
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						Deploy: latest.DeployConfig{
							KubeContext: "staging",
						},
					}},
					latest.Profile{
						Name:       "prod",
						Activation: []latest.Activation{{KubeContext: "prod-context"}},
					},
				),
			),
			shouldErr: true,
		},
		{
			description:              "activate kubecontext with kubecontext override",
			profile:                  "profile",
			profileAutoActivationCli: true,
			config: config(
				withProfiles(latest.Profile{
					Name: "profile",
					Pipeline: latest.Pipeline{
						Deploy: latest.DeployConfig{
							KubeContext: "staging",
						},
					}},
				),
			),
			expected: config(
				withKubeContext("staging"),
			),
		},
		{
			description:              "when CLI flag is given, profiles with conflicting kube-context produce no error",
			profile:                  "profile",
			profileAutoActivationCli: true,
			config: config(
				withProfiles(
					latest.Profile{
						Name:       "prod",
						Activation: []latest.Activation{{KubeContext: "prod-context"}},
					},
					latest.Profile{
						Name: "profile",
						Pipeline: latest.Pipeline{
							Deploy: latest.DeployConfig{
								KubeContext: "staging",
							},
						}},
				),
			),
			kubeContextCli: "prod-context",
			expected: config(
				withKubeContext("staging"),
			),
		},
		{
			description:              "Profile auto activated with profile auto activation cli set to true",
			profile:                  "dev",
			profileAutoActivationCli: true,
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "dev",
				},
					latest.Profile{
						Name:       "prod",
						Activation: []latest.Activation{{KubeContext: "prod-context"}},
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
			description:              "Profile not auto activated with profile auto activation cli set to false",
			profile:                  "dev",
			profileAutoActivationCli: false,
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "dev",
				},
					latest.Profile{
						Name:       "prod",
						Activation: []latest.Activation{{KubeContext: "prod-context"}},
						Pipeline: latest.Pipeline{
							Build: latest.BuildConfig{
								TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
							},
						},
					}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			setupFakeKubeConfig(t, api.Config{CurrentContext: "prod-context"})
			err := ApplyProfiles(test.config, cfg.SkaffoldOptions{
				Profiles:              []string{test.profile},
				KubeContext:           test.kubeContextCli,
				ProfileAutoActivation: test.profileAutoActivationCli,
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
		envs        map[string]string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "Selected on the command line",
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
				Command:               "dev",
				Profiles:              []string{"activated", "also-activated"},
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
				ProfileAutoActivation: true,
				Command:               "dev",
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
			envs:        map[string]string{"KEY": "VALUE"},
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
			},
			profiles: []latest.Profile{
				{Name: "activated", Activation: []latest.Activation{{Env: "KEY=VALUE"}}},
				{Name: "not-activated", Activation: []latest.Activation{{Env: "KEY=OTHER"}}},
				{Name: "also-activated", Activation: []latest.Activation{{Env: "KEY=!OTHER"}}},
				{Name: "not-treated-as-regex", Activation: []latest.Activation{{Env: "KEY="}}},
				{Name: "regex-activated", Activation: []latest.Activation{{Env: "KEY=V.*E"}}},
				{Name: "regex-activated-two", Activation: []latest.Activation{{Env: "KEY=^V.*E$"}}},
				{Name: "regex-activated-substring-match", Activation: []latest.Activation{{Env: "KEY=^VAL"}}},
			},
			expected: []string{"activated", "also-activated", "regex-activated", "regex-activated-two", "regex-activated-substring-match"},
		}, {
			description: "Invalid env variable",
			envs:        map[string]string{"KEY": "VALUE"},
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
			},
			profiles: []latest.Profile{
				{Name: "activated", Activation: []latest.Activation{{Env: "KEY:VALUE"}}},
			},
			shouldErr: true,
		}, {
			description: "Auto-activated by kube context",
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
			},
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
			envs:        map[string]string{"KEY": "VALUE"},
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
				Command:               "dev",
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
				ProfileAutoActivation: true,
				Command:               "dev",
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
		{
			description: "Activation for undefined environment variable and empty value",
			envs:        map[string]string{"ABC": ""},
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
				Command:               "dev",
			},
			profiles: []latest.Profile{
				{
					Name: "empty", Activation: []latest.Activation{{
						Env: "ABC=",
					}},
				},
				{
					Name: "empty-by-regex", Activation: []latest.Activation{{
						Env: "ABC=^$",
					}},
				},
				{
					Name: "not-empty", Activation: []latest.Activation{{
						Env: "ABC=!",
					}},
				},
				{
					Name: "one", Activation: []latest.Activation{{
						Env: "ABC=1",
					}},
				},
				{
					Name: "not-one", Activation: []latest.Activation{{
						Env: "ABC=!1",
					}},
				},
				{
					Name: "two", Activation: []latest.Activation{{
						Env: "ABC=2",
					}},
				},
			},
			expected: []string{"empty", "empty-by-regex", "not-one"},
		},
		{
			description: "Activation for filled environment variable",
			envs:        map[string]string{"ABC": "1"},
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
				Command:               "dev",
			},
			profiles: []latest.Profile{
				{
					Name: "empty", Activation: []latest.Activation{{
						Env: "ABC=",
					}},
				},
				{
					Name: "one", Activation: []latest.Activation{{
						Env: "ABC=1",
					}},
				},
				{
					Name: "one-as-well", Activation: []latest.Activation{{
						Command: "not-triggered",
					}, {
						Env: "ABC=1",
					}},
				},
				{
					Name: "two", Activation: []latest.Activation{{
						Command: "build",
					}, {
						Env: "ABC=2",
					}},
				},
				{
					Name: "not-two", Activation: []latest.Activation{{
						Command: "build",
					}, {
						Env: "ABC=!2",
					}},
				},
			},
			expected: []string{"one", "one-as-well", "not-two"},
		},
		{
			description: "Profiles on the command line are activated after auto-activated profiles",
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
				Command:               "run",
				Profiles:              []string{"activated", "also-activated"},
			},
			profiles: []latest.Profile{
				{Name: "run-profile", Activation: []latest.Activation{{Command: "run"}}},
			},
			expected: []string{"run-profile", "activated", "also-activated"},
		},
		{
			description: "Selected on the command line with auto activation disabled",
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: false,
				Command:               "dev",
				Profiles:              []string{"activated", "also-activated"},
			},
			profiles: []latest.Profile{
				{Name: "activated"},
				{Name: "not-activated"},
				{Name: "also-activated"},
				{Name: "not-activated-regexp", Activation: []latest.Activation{{KubeContext: "prod-.*"}}},
				{Name: "not-activated-kubecontext", Activation: []latest.Activation{{KubeContext: "prod-context"}}},
			},
			expected: []string{"activated", "also-activated"},
		},
		{
			description: "Disabled on the command line",
			opts: cfg.SkaffoldOptions{
				ProfileAutoActivation: true,
				Command:               "dev",
				Profiles:              []string{"-dev-profile"},
			},
			profiles: []latest.Profile{
				{Name: "dev-profile", Activation: []latest.Activation{{Command: "dev"}}},
				{Name: "run-or-dev-profile", Activation: []latest.Activation{{Command: "(run)|(dev)"}}},
			},
			expected: []string{"run-or-dev-profile"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetEnvs(test.envs)
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "prod-context"})

			activated, _, err := activatedProfiles(test.profiles, test.opts)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, activated)
		})
	}
}

func TestYamlAlias(t *testing.T) {
	config := `
.activation_common: &activation_common
  env: ABC=common
build:
  artifacts:
  - image: example
profiles:
- name: simple1
  activation:
  - *activation_common
  - env: ABC=1
  build:
    artifacts:
    - image: simpleimage1
- name: simple2
  activation:
  - *activation_common
  - env: ABC=2
  build:
    artifacts:
    - image: simpleimage2
`

	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("skaffold.yaml", addVersion(config))

		parsed, err := ParseConfig(tmpDir.Path("skaffold.yaml"))
		t.RequireNoError(err)

		skaffoldConfig := parsed.(*latest.SkaffoldConfig)

		t.CheckDeepEqual(2, len(skaffoldConfig.Profiles))
		t.CheckDeepEqual("simple1", skaffoldConfig.Profiles[0].Name)
		t.CheckDeepEqual([]latest.Activation{{Env: "ABC=common"}, {Env: "ABC=1"}}, skaffoldConfig.Profiles[0].Activation)
		t.CheckDeepEqual("simple2", skaffoldConfig.Profiles[1].Name)
		t.CheckDeepEqual([]latest.Activation{{Env: "ABC=common"}, {Env: "ABC=2"}}, skaffoldConfig.Profiles[1].Activation)

		err = ApplyProfiles(skaffoldConfig, cfg.SkaffoldOptions{
			Profiles: []string{"simple1"},
		})
		t.CheckNoError(err)

		t.CheckDeepEqual(1, len(skaffoldConfig.Build.Artifacts))
		t.CheckDeepEqual(latest.Artifact{ImageName: "simpleimage1"}, *skaffoldConfig.Build.Artifacts[0])
	})
}

func str(value string) *interface{} {
	var v interface{} = value
	return &v
}

func addVersion(yaml string) string {
	return fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latest.Version, yaml)
}

func setupFakeKubeConfig(t *testutil.T, config api.Config) {
	t.Override(&kubectx.CurrentConfig, func() (api.Config, error) {
		return config, nil
	})
}
