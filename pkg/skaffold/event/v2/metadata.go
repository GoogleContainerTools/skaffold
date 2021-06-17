/*
Copyright 2021 The Skaffold Authors

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

package v2

import (
	"fmt"
	"strings"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func LogMetaEvent() {
	metadata := handler.state.Metadata
	handler.handle(
		&proto.Event{
			EventType: &proto.Event_MetaEvent{
				MetaEvent: &proto.MetaEvent{
					Entry:    fmt.Sprintf("Starting Skaffold: %+v", version.Get()),
					Metadata: metadata,
				},
			},
		},
	)
}

func initializeMetadata(pipelines []latestV1.Pipeline, kubeContext string, runID string) *proto.Metadata {
	m := &proto.Metadata{
		Build:  &proto.BuildMetadata{},
		Deploy: &proto.DeployMetadata{},
		RunID:  runID,
	}

	// TODO: Event metadata should support multiple build types.
	// All pipelines are currently constrained to have the same build type.
	switch {
	case pipelines[0].Build.LocalBuild != nil:
		m.Build.Type = proto.BuildType_LOCAL
	case pipelines[0].Build.GoogleCloudBuild != nil:
		m.Build.Type = proto.BuildType_GCB
	case pipelines[0].Build.Cluster != nil:
		m.Build.Type = proto.BuildType_CLUSTER
	default:
		m.Build.Type = proto.BuildType_UNKNOWN_BUILD_TYPE
	}

	var artifacts []*proto.BuildMetadata_Artifact
	var deployers []*proto.DeployMetadata_Deployer
	for _, p := range pipelines {
		artifacts = append(artifacts, getArtifacts(p.Build)...)
		deployers = append(deployers, getDeploy(p.Deploy)...)
	}
	m.Build.Artifacts = artifacts

	if len(deployers) == 0 {
		m.Deploy = &proto.DeployMetadata{}
	} else {
		m.Deploy = &proto.DeployMetadata{
			Deployers: deployers,
			Cluster:   getClusterType(kubeContext),
		}
	}
	return m
}

func getArtifacts(b latestV1.BuildConfig) []*proto.BuildMetadata_Artifact {
	result := []*proto.BuildMetadata_Artifact{}
	for _, a := range b.Artifacts {
		artifact := &proto.BuildMetadata_Artifact{
			Name: a.ImageName,
		}
		switch {
		case a.BazelArtifact != nil:
			artifact.Type = proto.BuilderType_BAZEL
		case a.BuildpackArtifact != nil:
			artifact.Type = proto.BuilderType_BUILDPACKS
		case a.CustomArtifact != nil:
			artifact.Type = proto.BuilderType_CUSTOM
		case a.DockerArtifact != nil:
			artifact.Type = proto.BuilderType_DOCKER
		case a.JibArtifact != nil:
			artifact.Type = proto.BuilderType_JIB
		case a.KanikoArtifact != nil:
			artifact.Type = proto.BuilderType_KANIKO
		default:
			artifact.Type = proto.BuilderType_UNKNOWN_BUILDER_TYPE
		}
		result = append(result, artifact)
	}
	return result
}

func getDeploy(d latestV1.DeployConfig) []*proto.DeployMetadata_Deployer {
	var deployers []*proto.DeployMetadata_Deployer

	if d.HelmDeploy != nil {
		deployers = append(deployers, &proto.DeployMetadata_Deployer{Type: proto.DeployerType_HELM, Count: int32(len(d.HelmDeploy.Releases))})
	}
	if d.KubectlDeploy != nil {
		deployers = append(deployers, &proto.DeployMetadata_Deployer{Type: proto.DeployerType_KUBECTL, Count: 1})
	}
	if d.KustomizeDeploy != nil {
		deployers = append(deployers, &proto.DeployMetadata_Deployer{Type: proto.DeployerType_KUSTOMIZE, Count: 1})
	}
	return deployers
}

func getClusterType(c string) proto.ClusterType {
	switch {
	case strings.Contains(c, "minikube"):
		return proto.ClusterType_MINIKUBE
	case strings.HasPrefix(c, "gke"):
		return proto.ClusterType_GKE
	default:
		return proto.ClusterType_OTHER
	}
}
