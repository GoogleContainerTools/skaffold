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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestSetDefaults(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ImageName: "first",
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
				},
			},
		},
	}

	err := Set(cfg)

	testutil.CheckError(t, false, err)

	testutil.CheckDeepEqual(t, "first", cfg.Build.Artifacts[0].ImageName)
	testutil.CheckDeepEqual(t, ".", cfg.Build.Artifacts[0].Workspace)
	testutil.CheckDeepEqual(t, "Dockerfile", cfg.Build.Artifacts[0].DockerArtifact.DockerfilePath)

	testutil.CheckDeepEqual(t, "second", cfg.Build.Artifacts[1].ImageName)
	testutil.CheckDeepEqual(t, "folder", cfg.Build.Artifacts[1].Workspace)
	testutil.CheckDeepEqual(t, "Dockerfile.second", cfg.Build.Artifacts[1].DockerArtifact.DockerfilePath)

	testutil.CheckDeepEqual(t, "third", cfg.Build.Artifacts[2].ImageName)
	testutil.CheckDeepEqual(t, []string{"."}, cfg.Build.Artifacts[2].CustomArtifact.Dependencies.Paths)
	testutil.CheckDeepEqual(t, []string(nil), cfg.Build.Artifacts[2].CustomArtifact.Dependencies.Ignore)
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
					},
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{},
					},
				},
			},
		}
		err := Set(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual("ns", cfg.Build.Cluster.Namespace)
		t.CheckDeepEqual(constants.DefaultKanikoTimeout, cfg.Build.Cluster.Timeout)

		// artifact types
		t.CheckDeepEqual(true, cfg.Pipeline.Build.Artifacts[0].KanikoArtifact != nil)
		t.CheckDeepEqual(true, cfg.Pipeline.Build.Artifacts[1].KanikoArtifact != nil)
		t.CheckDeepEqual(false, cfg.Pipeline.Build.Artifacts[2].KanikoArtifact != nil)

		// pull secret set
		cfg = &latest.SkaffoldConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							PullSecret: "path/to/pull/secret",
						},
					},
				},
			},
		}
		err = Set(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual(constants.DefaultKanikoSecretName, cfg.Build.Cluster.PullSecretName)
		t.CheckDeepEqual(constants.DefaultKanikoSecretMountPath, cfg.Build.Cluster.PullSecretMountPath)

		// pull secret mount path set
		path := "/path"
		cfg = &latest.SkaffoldConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							PullSecret:          "path/to/pull/secret",
							PullSecretMountPath: path,
						},
					},
				},
			},
		}

		err = Set(cfg)
		t.CheckNoError(err)
		t.CheckDeepEqual(constants.DefaultKanikoSecretName, cfg.Build.Cluster.PullSecretName)
		t.CheckDeepEqual(path, cfg.Build.Cluster.PullSecretMountPath)

		// default docker config
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latest.DockerConfig{}
		err = Set(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual(constants.DefaultKanikoDockerConfigSecretName, cfg.Build.Cluster.DockerConfig.SecretName)

		// docker config with path
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latest.DockerConfig{
			Path: "/path",
		}
		err = Set(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual("docker-cfg", cfg.Build.Cluster.DockerConfig.SecretName)
		t.CheckDeepEqual("/path", cfg.Build.Cluster.DockerConfig.Path)

		// docker config with secret name
		cfg.Pipeline.Build.BuildType.Cluster.DockerConfig = &latest.DockerConfig{
			SecretName: "secret",
		}
		err = Set(cfg)

		t.CheckNoError(err)
		t.CheckDeepEqual("secret", cfg.Build.Cluster.DockerConfig.SecretName)
		t.CheckDeepEqual("", cfg.Build.Cluster.DockerConfig.Path)
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

	err := Set(cfg)

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

	err := Set(cfg)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, constants.DefaultCloudBuildDockerImage, cfg.Build.GoogleCloudBuild.DockerImage)
	testutil.CheckDeepEqual(t, constants.DefaultCloudBuildMavenImage, cfg.Build.GoogleCloudBuild.MavenImage)
	testutil.CheckDeepEqual(t, constants.DefaultCloudBuildGradleImage, cfg.Build.GoogleCloudBuild.GradleImage)
}

func TestSetDefaultPortForwardNamespace(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{},
			PortForward: []*latest.PortForwardResource{
				{
					Type:      constants.Service,
					Namespace: "mynamespace",
				}, {
					Type: constants.Service,
				},
			},
		},
	}
	err := Set(cfg)
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "mynamespace", cfg.PortForward[0].Namespace)
	testutil.CheckDeepEqual(t, constants.DefaultPortForwardNamespace, cfg.PortForward[1].Namespace)
}

func TestSetPortForwardLocalPort(t *testing.T) {
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{},
			PortForward: []*latest.PortForwardResource{
				{
					Type: constants.Service,
					Port: 8080,
				}, {
					Type:      constants.Service,
					Port:      8080,
					LocalPort: 9000,
				},
			},
		},
	}
	err := Set(cfg)
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, 8080, cfg.PortForward[0].LocalPort)
	testutil.CheckDeepEqual(t, 9000, cfg.PortForward[1].LocalPort)
}
