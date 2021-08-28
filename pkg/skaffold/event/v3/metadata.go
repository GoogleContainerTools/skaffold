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

package v3

import (
	"fmt"
	"strings"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	protoV3 "github.com/GoogleContainerTools/skaffold/proto/v3"
)

func LogMetaEvent() {
	metadata := handler.state.Metadata
	metaEvent := &protoV3.MetaEvent{
		Entry:    fmt.Sprintf("Starting Skaffold: %+v", version.Get()),
		Metadata: metadata,
	}
	handler.handle("Id-Metadata", metaEvent, MetaEvent)
}

func initializeMetadata(pipelines []latestV1.Pipeline, kubeContext string, runID string) *protoV3.Metadata {
	m := &protoV3.Metadata{
		Build:  &protoV3.BuildMetadata{},
		Render: &protoV3.RenderMetadata{},
		Deploy: &protoV3.DeployMetadata{},
		RunID:  runID,
	}

	// TODO: Event metadata should support multiple build types.
	// All pipelines are currently constrained to have the same build type.
	switch {
	case pipelines[0].Build.LocalBuild != nil:
		m.Build.Type = protoV3.BuildType_LOCAL
	case pipelines[0].Build.GoogleCloudBuild != nil:
		m.Build.Type = protoV3.BuildType_GCB
	case pipelines[0].Build.Cluster != nil:
		m.Build.Type = protoV3.BuildType_CLUSTER
	default:
		m.Build.Type = protoV3.BuildType_UNKNOWN_BUILD_TYPE
	}

	var artifacts []*protoV3.BuildMetadata_Artifact
	var deployers []*protoV3.DeployMetadata_Deployer
	for _, p := range pipelines {
		artifacts = append(artifacts, getArtifacts(p.Build)...)
		deployers = append(deployers, getDeploy(p.Deploy)...)
	}
	m.Build.Artifacts = artifacts

	if len(deployers) == 0 {
		m.Deploy = &protoV3.DeployMetadata{}
	} else {
		m.Deploy = &protoV3.DeployMetadata{
			Deployers: deployers,
			Cluster:   getClusterType(kubeContext),
		}
	}
	// TODO(v2 render): Add the renderMetadata initialization once the pipeline is switched to latestV2.Pipeline
	return m
}

func getArtifacts(b latestV1.BuildConfig) []*protoV3.BuildMetadata_Artifact {
	result := []*protoV3.BuildMetadata_Artifact{}
	for _, a := range b.Artifacts {
		artifact := &protoV3.BuildMetadata_Artifact{
			Name:    a.ImageName,
			Context: a.Workspace,
		}
		switch {
		case a.BazelArtifact != nil:
			artifact.Type = protoV3.BuilderType_BAZEL
		case a.BuildpackArtifact != nil:
			artifact.Type = protoV3.BuilderType_BUILDPACKS
		case a.CustomArtifact != nil:
			artifact.Type = protoV3.BuilderType_CUSTOM
		case a.DockerArtifact != nil:
			artifact.Type = protoV3.BuilderType_DOCKER
			artifact.Dockerfile = a.DockerArtifact.DockerfilePath
		case a.JibArtifact != nil:
			artifact.Type = protoV3.BuilderType_JIB
		case a.KanikoArtifact != nil:
			artifact.Type = protoV3.BuilderType_KANIKO
			artifact.Dockerfile = a.KanikoArtifact.DockerfilePath
		default:
			artifact.Type = protoV3.BuilderType_UNKNOWN_BUILDER_TYPE
		}
		result = append(result, artifact)
	}
	return result
}

func getDeploy(d latestV1.DeployConfig) []*protoV3.DeployMetadata_Deployer {
	var deployers []*protoV3.DeployMetadata_Deployer

	if d.HelmDeploy != nil {
		deployers = append(deployers, &protoV3.DeployMetadata_Deployer{Type: protoV3.DeployerType_HELM, Count: int32(len(d.HelmDeploy.Releases))})
	}
	if d.KubectlDeploy != nil {
		deployers = append(deployers, &protoV3.DeployMetadata_Deployer{Type: protoV3.DeployerType_KUBECTL, Count: 1})
	}
	if d.KustomizeDeploy != nil {
		deployers = append(deployers, &protoV3.DeployMetadata_Deployer{Type: protoV3.DeployerType_KUSTOMIZE, Count: 1})
	}
	return deployers
}

func getClusterType(c string) protoV3.ClusterType {
	switch {
	case strings.Contains(c, "minikube"):
		return protoV3.ClusterType_MINIKUBE
	case strings.HasPrefix(c, "gke"):
		return protoV3.ClusterType_GKE
	default:
		return protoV3.ClusterType_OTHER
	}
}
