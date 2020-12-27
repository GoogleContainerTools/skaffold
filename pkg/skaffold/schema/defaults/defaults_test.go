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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetDefaults(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ImageName: "first",
						Dependencies: []*latest.ArtifactDependency{
							{ImageName: "second", Alias: "secondAlias"},
							{ImageName: "third"},
						},
					},
					{
						ImageName: "second",
						Workspace: "folder",
						ArtifactType: latest.ArtifactType{
							DockerArtifact: &latest.DockerArtifact{
								DockerfilePath: "Dockerfile.second",
							},
						},
					},
					{
						ImageName: "third",
						ArtifactType: latest.ArtifactType{
							CustomArtifact: &latest.CustomArtifact{},
						},
					},
					{
						ImageName: "fourth",
						ArtifactType: latest.ArtifactType{
							BuildpackArtifact: &latest.BuildpackArtifact{},
						},
						Sync: &latest.Sync{},
					},
					{
						ImageName: "fifth",
						ArtifactType: latest.ArtifactType{
							JibArtifact: &latest.JibArtifact{},
						},
						Sync: &latest.Sync{},
					},
					{
						ImageName: "sixth",
						ArtifactType: latest.ArtifactType{
							BuildpackArtifact: &latest.BuildpackArtifact{},
						},
					},
					{
						ImageName: "seventh",
						ArtifactType: latest.ArtifactType{
							BuildpackArtifact: &latest.BuildpackArtifact{},
						},
						Sync: &latest.Sync{Auto: util.BoolPtr(false)},
					},
				},
			},
		},
	}

	err := Set(cfg, true)

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
		cfg := &latest.SkaffoldConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					Artifacts: []*latest.Artifact{
						{
							ImageName: "docker",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{},
							},
						},
						{
							ImageName: "kaniko",
							ArtifactType: latest.ArtifactType{
								KanikoArtifact: &latest.KanikoArtifact{},
							},
						},
						{
							ImageName: "custom",
							ArtifactType: latest.ArtifactType{
								CustomArtifact: &latest.CustomArtifact{},
							},
						},
						{
							ImageName: "buildpacks",
							ArtifactType: latest.ArtifactType{
								BuildpackArtifact: &latest.BuildpackArtifact{},
							},
						},
					},
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{},
					},
				},
			},
		}
		err := Set(cfg, true)

		t.CheckNoError(err)
		t.CheckDeepEqual("ns", cfg.Build.Cluster.Namespace)
		t.CheckDeepEqual(kaniko.DefaultTimeout, cfg.Build.Cluster.Timeout)

		// artifact types
		t.CheckNotNil(cfg.Pipeline.Build.Artifacts[0].KanikoArtifact)
		t.CheckNotNil(cfg.Pipeline.Build.Artifacts[1].KanikoArtifact)
		t.CheckNil(cfg.Pipeline.Build.Artifacts[2].KanikoArtifact)
		t.CheckNil(cfg.Pipeline.Build.Artifacts[3].KanikoArtifact)

		// pull secret set
		cfg = &latest.SkaffoldConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							PullSecretPath: "path/to/pull/secret",
						},
					},
				},
			},
		}
		err = Set(cfg, true)

		t.CheckNoError(err)

		t.CheckDeepEqual(kaniko.DefaultSecretMountPath, cfg.Build.Cluster.PullSecretMountPath)

		// pull secret mount path set
		path := "/path"
		cfg = &latest.SkaffoldConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							PullSecretPath:      "path/to/pull/secret",
							PullSecretMountPath: path,
						},
					},
				},
			},
		}

		err = Set(cfg, true)
		t.CheckNoError(err)
		t.CheckDeepEqual(path, cfg.Build.Cluster.PullSecretMountPath)

		// default docker config
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latest.DockerConfig{}
		err = Set(cfg, true)

		t.CheckNoError(err)

		// docker config with path
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latest.DockerConfig{
			Path: "/path",
		}
		err = Set(cfg, true)

		t.CheckNoError(err)
		t.CheckDeepEqual("/path", cfg.Build.Cluster.DockerConfig.Path)

		// docker config with secret name
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latest.DockerConfig{
			SecretName: "secret",
		}
		err = Set(cfg, true)

		t.CheckNoError(err)
		t.CheckDeepEqual("secret", cfg.Build.Cluster.DockerConfig.SecretName)
		t.CheckEmpty(cfg.Build.Cluster.DockerConfig.Path)
	})
}

func TestCustomBuildWithCluster(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ImageName: "image",
						ArtifactType: latest.ArtifactType{
							CustomArtifact: &latest.CustomArtifact{
								BuildCommand: "./build.sh",
							},
						},
					},
				},
				BuildType: latest.BuildType{
					Cluster: &latest.ClusterDetails{},
				},
			},
		},
	}

	err := Set(cfg, true)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, (*latest.KanikoArtifact)(nil), cfg.Build.Artifacts[0].KanikoArtifact)
}

func TestSetDefaultsOnCloudBuild(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{ImageName: "image"},
				},
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{},
				},
			},
		},
	}

	err := Set(cfg, true)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, defaultCloudBuildDockerImage, cfg.Build.GoogleCloudBuild.DockerImage)
	testutil.CheckDeepEqual(t, defaultCloudBuildMavenImage, cfg.Build.GoogleCloudBuild.MavenImage)
	testutil.CheckDeepEqual(t, defaultCloudBuildGradleImage, cfg.Build.GoogleCloudBuild.GradleImage)
	testutil.CheckDeepEqual(t, defaultCloudBuildPackImage, cfg.Build.GoogleCloudBuild.PackImage)
}

func TestSetDefaultsOnLocalBuild(t *testing.T) {
	cfg := &latest.SkaffoldConfig{}

	err := Set(cfg, true)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, 1, *cfg.Build.LocalBuild.Concurrency)
}

func TestSetPortForwardLocalPort(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{},
			PortForward: []*latest.PortForwardResource{
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
	err := Set(cfg, true)
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, 8080, cfg.PortForward[0].LocalPort)
	testutil.CheckDeepEqual(t, 9000, cfg.PortForward[1].LocalPort)
}

func TestSetDefaultPortForwardAddress(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{},
			PortForward: []*latest.PortForwardResource{
				{
					Type:    constants.Service,
					Address: "0.0.0.0",
				}, {
					Type: constants.Service,
				},
			},
		},
	}
	err := Set(cfg, true)
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "0.0.0.0", cfg.PortForward[0].Address)
	testutil.CheckDeepEqual(t, constants.DefaultPortForwardAddress, cfg.PortForward[1].Address)
}

func TestSetLogsConfig(t *testing.T) {
	tests := []struct {
		description string
		input       latest.LogsConfig
		expected    latest.LogsConfig
	}{
		{
			description: "prefix defaults to 'container'",
			input:       latest.LogsConfig{},
			expected: latest.LogsConfig{
				Prefix: "container",
			},
		},
		{
			description: "don't override existing prefix",
			input: latest.LogsConfig{
				Prefix: "none",
			},
			expected: latest.LogsConfig{
				Prefix: "none",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Deploy: latest.DeployConfig{
						Logs: test.input,
					},
				},
			}

			err := Set(&cfg, true)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, cfg.Deploy.Logs)
		})
	}
}
