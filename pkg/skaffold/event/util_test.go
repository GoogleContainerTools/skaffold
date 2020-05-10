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

package event

import (
	"sort"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEmptyState(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest.Pipeline
		cluster     string
		expected    *proto.Metadata
	}{
		{
			description: "one build artifact minikube cluster multiple deployers",
			cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}},
					Artifacts: []*latest.Artifact{{ImageName: "img", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}}},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
						HelmDeploy:    &latest.HelmDeploy{Releases: []latest.HelmRelease{{Name: "first"}, {Name: "second"}}},
					},
				},
			},
			cluster: "minikube",
			expected: &proto.Metadata{
				Build: &proto.BuildMetadata{
					NumberOfArtifacts: 1,
					Type:              proto.BuildType_LOCAL,
					Builders:          []*proto.BuildMetadata_ImageBuilder{{Type: proto.BuilderType_DOCKER, Count: 1}},
				},
				Deploy: &proto.DeployMetadata{
					Cluster: proto.ClusterType_MINIKUBE,
					Deployers: []*proto.DeployMetadata_Deployer{
						{Type: proto.DeployerType_HELM, Count: 2},
						{Type: proto.DeployerType_KUBECTL, Count: 1},
					}},
			},
		},
		{
			description: "multiple artifacts of different types gke cluster 1 deployer ",
			cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}},
					Artifacts: []*latest.Artifact{
						{ImageName: "img1", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
						{ImageName: "img2", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
						{ImageName: "img3", ArtifactType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{}}},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KustomizeDeploy: &latest.KustomizeDeploy{},
					},
				},
			},
			cluster: "gke-tejal-test",
			expected: &proto.Metadata{
				Build: &proto.BuildMetadata{
					NumberOfArtifacts: 3,
					Type:              proto.BuildType_CLUSTER,
					Builders: []*proto.BuildMetadata_ImageBuilder{
						{Type: proto.BuilderType_JIB, Count: 1},
						{Type: proto.BuilderType_DOCKER, Count: 2},
					},
				},
				Deploy: &proto.DeployMetadata{
					Cluster:   proto.ClusterType_GKE,
					Deployers: []*proto.DeployMetadata_Deployer{{Type: proto.DeployerType_KUSTOMIZE, Count: 1}}},
			},
		},
		{
			description: "no deployer, kaniko artifact, GCB build",
			cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{}},
					Artifacts: []*latest.Artifact{
						{ImageName: "img1", ArtifactType: latest.ArtifactType{KanikoArtifact: &latest.KanikoArtifact{}}},
					},
				},
			},
			cluster: "gke-tejal-test",
			expected: &proto.Metadata{
				Build: &proto.BuildMetadata{
					NumberOfArtifacts: 1,
					Type:              proto.BuildType_GCB,
					Builders:          []*proto.BuildMetadata_ImageBuilder{{Type: proto.BuilderType_KANIKO, Count: 1}},
				},
				Deploy: &proto.DeployMetadata{},
			},
		},
		{
			description: "no build, kustomize deployer other cluster",
			cfg: latest.Pipeline{
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KustomizeDeploy: &latest.KustomizeDeploy{},
					},
				},
			},
			cluster: "some-private",
			expected: &proto.Metadata{
				Build: &proto.BuildMetadata{
					Builders: []*proto.BuildMetadata_ImageBuilder{},
				},
				Deploy: &proto.DeployMetadata{
					Cluster:   proto.ClusterType_OTHER,
					Deployers: []*proto.DeployMetadata_Deployer{{Type: proto.DeployerType_KUSTOMIZE, Count: 1}},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			handler = &eventHandler{
				state: emptyState(test.cfg, test.cluster, true, true, true),
			}
			metadata := handler.state.Metadata
			builders := metadata.Build.Builders

			// sort and compare
			sort.Slice(builders, func(i, j int) bool { return builders[i].Type < builders[j].Type })
			t.CheckDeepEqual(metadata, test.expected)
		})
	}
}
