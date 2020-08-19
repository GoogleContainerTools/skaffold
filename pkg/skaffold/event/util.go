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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func initializeMetadata(p latest.Pipeline, kubeContext string) *proto.Metadata {
	m := &proto.Metadata{
		Build: &proto.BuildMetadata{
			NumberOfArtifacts: int32(len(p.Build.Artifacts)),
		},
		Deploy: &proto.DeployMetadata{},
	}

	switch {
	case p.Build.LocalBuild != nil:
		m.Build.Type = proto.BuildType_LOCAL
	case p.Build.GoogleCloudBuild != nil:
		m.Build.Type = proto.BuildType_GCB
	case p.Build.Cluster != nil:
		m.Build.Type = proto.BuildType_CLUSTER
	default:
		m.Build.Type = proto.BuildType_UNKNOWN_BUILD_TYPE
	}

	m.Build.Builders = getBuilders(p.Build)
	m.Deploy = getDeploy(p.Deploy, kubeContext)
	return m
}

func getBuilders(b latest.BuildConfig) []*proto.BuildMetadata_ImageBuilder {
	m := map[proto.BuilderType]int{}
	for _, a := range b.Artifacts {
		switch {
		case a.BazelArtifact != nil:
			updateOrAddKey(m, proto.BuilderType_BAZEL)
		case a.BuildpackArtifact != nil:
			updateOrAddKey(m, proto.BuilderType_BUILDPACKS)
		case a.CustomArtifact != nil:
			updateOrAddKey(m, proto.BuilderType_CUSTOM)
		case a.DockerArtifact != nil:
			updateOrAddKey(m, proto.BuilderType_DOCKER)
		case a.JibArtifact != nil:
			updateOrAddKey(m, proto.BuilderType_JIB)
		case a.KanikoArtifact != nil:
			updateOrAddKey(m, proto.BuilderType_KANIKO)
		default:
			updateOrAddKey(m, proto.BuilderType_UNKNOWN_BUILDER_TYPE)
		}
	}
	builders := make([]*proto.BuildMetadata_ImageBuilder, len(m))
	i := 0
	for k, v := range m {
		builders[i] = &proto.BuildMetadata_ImageBuilder{Type: k, Count: int32(v)}
		i++
	}
	return builders
}

func getDeploy(d latest.DeployConfig, c string) *proto.DeployMetadata {
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
	if len(deployers) == 0 {
		return &proto.DeployMetadata{}
	}

	return &proto.DeployMetadata{
		Deployers: deployers,
		Cluster:   getClusterType(c),
	}
}

func updateOrAddKey(m map[proto.BuilderType]int, k proto.BuilderType) {
	if _, ok := m[k]; ok {
		m[k]++
		return
	}
	m[k] = 1
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
