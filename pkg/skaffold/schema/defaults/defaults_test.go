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

package defaults

import (
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetDefaults(t *testing.T) {
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{
				Artifacts: []*latestV2.Artifact{
					{
						ImageName: "first",
						Dependencies: []*latestV2.ArtifactDependency{
							{ImageName: "second", Alias: "secondAlias"},
							{ImageName: "third"},
						},
					},
					{
						ImageName: "second",
						Workspace: "folder",
						ArtifactType: latestV2.ArtifactType{
							DockerArtifact: &latestV2.DockerArtifact{
								DockerfilePath: "Dockerfile.second",
							},
						},
					},
					{
						ImageName: "third",
						ArtifactType: latestV2.ArtifactType{
							CustomArtifact: &latestV2.CustomArtifact{},
						},
					},
					{
						ImageName: "fourth",
						ArtifactType: latestV2.ArtifactType{
							BuildpackArtifact: &latestV2.BuildpackArtifact{},
						},
						Sync: &latestV2.Sync{},
					},
					{
						ImageName: "fifth",
						ArtifactType: latestV2.ArtifactType{
							JibArtifact: &latestV2.JibArtifact{},
						},
						Sync: &latestV2.Sync{},
					},
					{
						ImageName: "sixth",
						ArtifactType: latestV2.ArtifactType{
							BuildpackArtifact: &latestV2.BuildpackArtifact{},
						},
					},
					{
						ImageName: "seventh",
						ArtifactType: latestV2.ArtifactType{
							BuildpackArtifact: &latestV2.BuildpackArtifact{},
						},
						Sync: &latestV2.Sync{Auto: util.BoolPtr(false)},
					},
				},
			},
		},
	}

	err := Set(cfg)
	SetDefaultDeployer(cfg)

	testutil.CheckError(t, false, err)

	testutil.CheckDeepEqual(t, "first", cfg.Build.Artifacts[0].ImageName)
	testutil.CheckDeepEqual(t, ".", cfg.Build.Artifacts[0].Workspace)
	testutil.CheckDeepEqual(t, "Dockerfile", cfg.Build.Artifacts[0].DockerArtifact.DockerfilePath)
	testutil.CheckDeepEqual(t, "secondAlias", cfg.Build.Artifacts[0].Dependencies[0].Alias)
	testutil.CheckDeepEqual(t, "third", cfg.Build.Artifacts[0].Dependencies[1].Alias)

	testutil.CheckDeepEqual(t, "second", cfg.Build.Artifacts[1].ImageName)
	testutil.CheckDeepEqual(t, "folder", cfg.Build.Artifacts[1].Workspace)
	testutil.CheckDeepEqual(t, "Dockerfile.second", cfg.Build.Artifacts[1].DockerArtifact.DockerfilePath)

	testutil.CheckDeepEqual(t, "third", cfg.Build.Artifacts[2].ImageName)
	testutil.CheckDeepEqual(t, []string{"."}, cfg.Build.Artifacts[2].CustomArtifact.Dependencies.Paths)
	testutil.CheckDeepEqual(t, []string(nil), cfg.Build.Artifacts[2].CustomArtifact.Dependencies.Ignore)

	testutil.CheckDeepEqual(t, "fourth", cfg.Build.Artifacts[3].ImageName)
	testutil.CheckDeepEqual(t, []string{"."}, cfg.Build.Artifacts[3].BuildpackArtifact.Dependencies.Paths)
	testutil.CheckDeepEqual(t, []string(nil), cfg.Build.Artifacts[3].BuildpackArtifact.Dependencies.Ignore)
	testutil.CheckDeepEqual(t, "project.toml", cfg.Build.Artifacts[3].BuildpackArtifact.ProjectDescriptor)
	testutil.CheckDeepEqual(t, util.BoolPtr(true), cfg.Build.Artifacts[3].Sync.Auto)

	testutil.CheckDeepEqual(t, "fifth", cfg.Build.Artifacts[4].ImageName)
	testutil.CheckDeepEqual(t, util.BoolPtr(true), cfg.Build.Artifacts[4].Sync.Auto)

	testutil.CheckDeepEqual(t, "sixth", cfg.Build.Artifacts[5].ImageName)
	testutil.CheckDeepEqual(t, []string{"."}, cfg.Build.Artifacts[5].BuildpackArtifact.Dependencies.Paths)
	testutil.CheckDeepEqual(t, []string(nil), cfg.Build.Artifacts[5].BuildpackArtifact.Dependencies.Ignore)
	testutil.CheckDeepEqual(t, "project.toml", cfg.Build.Artifacts[5].BuildpackArtifact.ProjectDescriptor)
	testutil.CheckDeepEqual(t, util.BoolPtr(true), cfg.Build.Artifacts[5].Sync.Auto)

	testutil.CheckDeepEqual(t, "seventh", cfg.Build.Artifacts[6].ImageName)
	testutil.CheckDeepEqual(t, []string{"."}, cfg.Build.Artifacts[6].BuildpackArtifact.Dependencies.Paths)
	testutil.CheckDeepEqual(t, []string(nil), cfg.Build.Artifacts[6].BuildpackArtifact.Dependencies.Ignore)
	testutil.CheckDeepEqual(t, "project.toml", cfg.Build.Artifacts[6].BuildpackArtifact.ProjectDescriptor)
	testutil.CheckDeepEqual(t, util.BoolPtr(false), cfg.Build.Artifacts[6].Sync.Auto)
}

func TestSetDefaultsOnCluster(t *testing.T) {
	testutil.Run(t, "no docker config", func(t *testutil.T) {
		t.SetupFakeKubernetesContext(api.Config{
			CurrentContext: "cluster1",
			Contexts: map[string]*api.Context{
				"cluster1": {Namespace: "ns"},
			},
		})

		// no docker config
		cfg := &latestV2.SkaffoldConfig{
			Pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					Artifacts: []*latestV2.Artifact{
						{
							ImageName: "docker",
							ArtifactType: latestV2.ArtifactType{
								DockerArtifact: &latestV2.DockerArtifact{},
							},
						},
						{
							ImageName: "kaniko",
							ArtifactType: latestV2.ArtifactType{
								KanikoArtifact: &latestV2.KanikoArtifact{},
							},
						},
						{
							ImageName: "custom",
							ArtifactType: latestV2.ArtifactType{
								CustomArtifact: &latestV2.CustomArtifact{},
							},
						},
						{
							ImageName: "buildpacks",
							ArtifactType: latestV2.ArtifactType{
								BuildpackArtifact: &latestV2.BuildpackArtifact{},
							},
						},
					},
					BuildType: latestV2.BuildType{
						Cluster: &latestV2.ClusterDetails{},
					},
				},
			},
		}
		err := Set(cfg)
		SetDefaultDeployer(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual("ns", cfg.Build.Cluster.Namespace)
		t.CheckDeepEqual(kaniko.DefaultTimeout, cfg.Build.Cluster.Timeout)

		// artifact types
		t.CheckNotNil(cfg.Pipeline.Build.Artifacts[0].KanikoArtifact)
		t.CheckNotNil(cfg.Pipeline.Build.Artifacts[1].KanikoArtifact)
		t.CheckNil(cfg.Pipeline.Build.Artifacts[2].KanikoArtifact)
		t.CheckNil(cfg.Pipeline.Build.Artifacts[3].KanikoArtifact)

		// pull secret set
		cfg = &latestV2.SkaffoldConfig{
			Pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					BuildType: latestV2.BuildType{
						Cluster: &latestV2.ClusterDetails{
							PullSecretPath: "path/to/pull/secret",
						},
					},
				},
			},
		}
		err = Set(cfg)
		SetDefaultDeployer(cfg)

		t.CheckNoError(err)

		t.CheckDeepEqual(kaniko.DefaultSecretMountPath, cfg.Build.Cluster.PullSecretMountPath)

		// pull secret mount path set
		path := "/path"
		cfg = &latestV2.SkaffoldConfig{
			Pipeline: latestV2.Pipeline{
				Build: latestV2.BuildConfig{
					BuildType: latestV2.BuildType{
						Cluster: &latestV2.ClusterDetails{
							PullSecretPath:      "path/to/pull/secret",
							PullSecretMountPath: path,
						},
					},
				},
			},
		}

		err = Set(cfg)
		SetDefaultDeployer(cfg)
		t.CheckNoError(err)
		t.CheckDeepEqual(path, cfg.Build.Cluster.PullSecretMountPath)

		// default docker config
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latestV2.DockerConfig{}
		err = Set(cfg)
		SetDefaultDeployer(cfg)

		t.CheckNoError(err)

		// docker config with path
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latestV2.DockerConfig{
			Path: "/path",
		}
		err = Set(cfg)
		SetDefaultDeployer(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual("/path", cfg.Build.Cluster.DockerConfig.Path)

		// docker config with secret name
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latestV2.DockerConfig{
			SecretName: "secret",
		}
		err = Set(cfg)
		SetDefaultDeployer(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual("secret", cfg.Build.Cluster.DockerConfig.SecretName)
		t.CheckEmpty(cfg.Build.Cluster.DockerConfig.Path)
	})
}

func TestCustomBuildWithCluster(t *testing.T) {
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{
				Artifacts: []*latestV2.Artifact{
					{
						ImageName: "image",
						ArtifactType: latestV2.ArtifactType{
							CustomArtifact: &latestV2.CustomArtifact{
								BuildCommand: "./build.sh",
							},
						},
					},
				},
				BuildType: latestV2.BuildType{
					Cluster: &latestV2.ClusterDetails{},
				},
			},
		},
	}

	err := Set(cfg)
	SetDefaultDeployer(cfg)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, (*latestV2.KanikoArtifact)(nil), cfg.Build.Artifacts[0].KanikoArtifact)
}

func TestSetDefaultsOnCloudBuild(t *testing.T) {
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{
				Artifacts: []*latestV2.Artifact{
					{ImageName: "image"},
				},
				BuildType: latestV2.BuildType{
					GoogleCloudBuild: &latestV2.GoogleCloudBuild{},
				},
			},
		},
	}

	err := Set(cfg)
	SetDefaultDeployer(cfg)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, defaultCloudBuildDockerImage, cfg.Build.GoogleCloudBuild.DockerImage)
	testutil.CheckDeepEqual(t, defaultCloudBuildMavenImage, cfg.Build.GoogleCloudBuild.MavenImage)
	testutil.CheckDeepEqual(t, defaultCloudBuildGradleImage, cfg.Build.GoogleCloudBuild.GradleImage)
	testutil.CheckDeepEqual(t, defaultCloudBuildPackImage, cfg.Build.GoogleCloudBuild.PackImage)
}

func TestSetDefaultsOnLocalBuild(t *testing.T) {
	cfg1 := &latestV2.SkaffoldConfig{Pipeline: latestV2.Pipeline{Build: latestV2.BuildConfig{}}}
	cfg2 := &latestV2.SkaffoldConfig{Pipeline: latestV2.Pipeline{Build: latestV2.BuildConfig{Artifacts: []*latestV2.Artifact{{ImageName: "foo"}}}}}

	err := Set(cfg1)
	testutil.CheckError(t, false, err)
	SetDefaultDeployer(cfg1)
	testutil.CheckDeepEqual(t, latestV2.LocalBuild{}, *cfg1.Build.LocalBuild)
	err = Set(cfg2)
	testutil.CheckError(t, false, err)
	SetDefaultDeployer(cfg2)
	testutil.CheckDeepEqual(t, 1, *cfg2.Build.LocalBuild.Concurrency)
}

func TestSetPortForwardLocalPort(t *testing.T) {
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{},
			PortForward: []*latestV2.PortForwardResource{
				{
					Type: constants.Service,
					Port: schemautil.FromInt(8080),
				}, {
					Type:      constants.Service,
					Port:      schemautil.FromInt(8080),
					LocalPort: 9000,
				},
			},
		},
	}
	err := Set(cfg)
	SetDefaultDeployer(cfg)
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, 8080, cfg.PortForward[0].LocalPort)
	testutil.CheckDeepEqual(t, 9000, cfg.PortForward[1].LocalPort)
}

func TestSetPortForwardOnEmptyPortForwardResource(t *testing.T) {
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{},
			PortForward: []*latestV2.PortForwardResource{
				nil,
			},
		},
	}
	err := Set(cfg)
	testutil.CheckError(t, true, err)
}

func TestSetDefaultPortForwardAddress(t *testing.T) {
	cfg := &latestV2.SkaffoldConfig{
		Pipeline: latestV2.Pipeline{
			Build: latestV2.BuildConfig{},
			PortForward: []*latestV2.PortForwardResource{
				{
					Type:    constants.Service,
					Address: "0.0.0.0",
				}, {
					Type: constants.Service,
				},
			},
		},
	}
	err := Set(cfg)
	SetDefaultDeployer(cfg)
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "0.0.0.0", cfg.PortForward[0].Address)
	testutil.CheckDeepEqual(t, constants.DefaultPortForwardAddress, cfg.PortForward[1].Address)
}

func TestSetDefaultDeployer(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latestV2.SkaffoldConfig
		expected    latestV2.DeployConfig
	}{
		{
			description: "no deployer definition",
			cfg:         &latestV2.SkaffoldConfig{},
			expected: latestV2.DeployConfig{
				DeployType: latestV2.DeployType{
					KubectlDeploy: &latestV2.KubectlDeploy{Manifests: []string{"k8s/*.yaml"}},
				},
			},
		},
		{
			description: "existing kubectl definition with local manifests",
			cfg: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Deploy: latestV2.DeployConfig{DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{Manifests: []string{"foo.yaml"}},
					}},
				},
			},
			expected: latestV2.DeployConfig{
				DeployType: latestV2.DeployType{
					KubectlDeploy: &latestV2.KubectlDeploy{Manifests: []string{"foo.yaml"}},
				},
			},
		},
		{
			description: "existing kubectl definition with remote manifests",
			cfg: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Deploy: latestV2.DeployConfig{DeployType: latestV2.DeployType{
						KubectlDeploy: &latestV2.KubectlDeploy{RemoteManifests: []string{"foo:bar"}},
					}},
				},
			},
			expected: latestV2.DeployConfig{
				DeployType: latestV2.DeployType{
					KubectlDeploy: &latestV2.KubectlDeploy{RemoteManifests: []string{"foo:bar"}},
				},
			},
		},
		{
			description: "existing helm definition",
			cfg: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Deploy: latestV2.DeployConfig{DeployType: latestV2.DeployType{
						HelmDeploy: &latestV2.HelmDeploy{Releases: []latestV2.HelmRelease{{ChartPath: "foo"}}},
					}},
				},
			},
			expected: latestV2.DeployConfig{
				DeployType: latestV2.DeployType{
					HelmDeploy: &latestV2.HelmDeploy{Releases: []latestV2.HelmRelease{{ChartPath: "foo"}}},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			SetDefaultDeployer(test.cfg)
			t.CheckDeepEqual(test.expected, test.cfg.Deploy)
		})
	}
}

func TestSetLogsConfig(t *testing.T) {
	tests := []struct {
		description string
		input       latestV2.LogsConfig
		expected    latestV2.LogsConfig
	}{
		{
			description: "prefix defaults to 'container'",
			input:       latestV2.LogsConfig{},
			expected: latestV2.LogsConfig{
				Prefix: "container",
			},
		},
		{
			description: "don't override existing prefix",
			input: latestV2.LogsConfig{
				Prefix: "none",
			},
			expected: latestV2.LogsConfig{
				Prefix: "none",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Deploy: latestV2.DeployConfig{
						Logs: test.input,
					},
				},
			}

			err := Set(&cfg)
			SetDefaultDeployer(&cfg)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, cfg.Deploy.Logs)
		})
	}
}
