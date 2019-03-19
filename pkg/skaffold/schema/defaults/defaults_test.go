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
	pipeline := &latest.SkaffoldPipeline{
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
			},
		},
	}

	err := Set(pipeline)

	testutil.CheckError(t, false, err)

	testutil.CheckDeepEqual(t, "first", pipeline.Build.Artifacts[0].ImageName)
	testutil.CheckDeepEqual(t, ".", pipeline.Build.Artifacts[0].Workspace)
	testutil.CheckDeepEqual(t, "Dockerfile", pipeline.Build.Artifacts[0].DockerArtifact.DockerfilePath)

	testutil.CheckDeepEqual(t, "second", pipeline.Build.Artifacts[1].ImageName)
	testutil.CheckDeepEqual(t, "folder", pipeline.Build.Artifacts[1].Workspace)
	testutil.CheckDeepEqual(t, "Dockerfile.second", pipeline.Build.Artifacts[1].DockerArtifact.DockerfilePath)
}

func TestSetDefaultsOnCluster(t *testing.T) {
	restore := testutil.SetupFakeKubernetesContext(t, api.Config{
		CurrentContext: "cluster1",
		Contexts: map[string]*api.Context{
			"cluster1": {Namespace: "ns"},
		},
	})
	defer restore()

	pipeline := &latest.SkaffoldPipeline{
		Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{
				{ImageName: "image"},
			},
			BuildType: latest.BuildType{
				Cluster: &latest.ClusterDetails{},
			},
		},
	}

	err := Set(pipeline)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "ns", pipeline.Build.Cluster.Namespace)
	testutil.CheckDeepEqual(t, constants.DefaultKanikoTimeout, pipeline.Build.Cluster.Timeout)
	testutil.CheckDeepEqual(t, constants.DefaultKanikoSecretName, pipeline.Build.Cluster.PullSecretName)
}

func TestSetDefaultsOnCloudBuild(t *testing.T) {
	pipeline := &latest.SkaffoldPipeline{
		Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{
				{ImageName: "image"},
			},
			BuildType: latest.BuildType{
				GoogleCloudBuild: &latest.GoogleCloudBuild{},
			},
		},
	}

	err := Set(pipeline)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, constants.DefaultCloudBuildDockerImage, pipeline.Build.GoogleCloudBuild.DockerImage)
	testutil.CheckDeepEqual(t, constants.DefaultCloudBuildMavenImage, pipeline.Build.GoogleCloudBuild.MavenImage)
	testutil.CheckDeepEqual(t, constants.DefaultCloudBuildGradleImage, pipeline.Build.GoogleCloudBuild.GradleImage)
}

func TestSetDefaultsOnPlugin(t *testing.T) {
	pipeline := &latest.SkaffoldPipeline{
		Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{
				{
					ImageName:     "image",
					BuilderPlugin: &latest.BuilderPlugin{},
				},
			},
		},
	}

	err := Set(pipeline)

	for _, a := range pipeline.Build.Artifacts {
		testutil.CheckDeepEqual(t, ".", a.Workspace)
	}
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, &latest.ExecutionEnvironment{
		Name:       constants.Local,
		Properties: map[string]interface{}{},
	}, pipeline.Build.ExecutionEnvironment)
}
