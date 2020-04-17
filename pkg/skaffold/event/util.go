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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func initializeMetadata(c latest.Pipeline, kc string) *proto.Metadata {
	m := &proto.Metadata{
		Build: &proto.BuildMetadata{
			NumberOfArtifacts: int32(len(c.Build.Artifacts)),
		},
		Deploy: &proto.DeployMetadata{},
	}

	switch {
	case c.Build.LocalBuild != nil:
		m.Build.Type = proto.BuildType_LOCAL
	case c.Build.GoogleCloudBuild != nil:
		m.Build.Type = proto.BuildType_GCB
	case c.Build.Cluster != nil:
		m.Build.Type = proto.BuildType_CLUSTER
	}

	m.Build.Builders = getBuilders(c.Build)
	m.Deploy.Deployers = getDeployers(c.Deploy)
	m.Deploy.Cluster = getClusterType(kc)
	return m
}

func getBuilders(b latest.BuildConfig) []*proto.BuildMetadata_Builders {
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
		}
	}
	builders := make([]*proto.BuildMetadata_Builders, len(m))
	i := 0
	for k, v := range m {
		builders[i] = &proto.BuildMetadata_Builders{Type: k, Count: int32(v)}
		i++
	}
	fmt.Println("    ", builders)
	return builders
}

func getDeployers(d latest.DeployConfig) []*proto.DeployMetadata_Deployers {
	m := map[proto.DeployerType]int{}
	if d.HelmDeploy != nil {
		m[proto.DeployerType_HELM] = len(d.HelmDeploy.Releases)
	}
	if d.KubectlDeploy != nil {
		m[proto.DeployerType_KUBECTL] = 1
	}
	if d.KustomizeDeploy != nil {
		m[proto.DeployerType_KUSTOMIZE] = 1
	}

	deployers := make([]*proto.DeployMetadata_Deployers, len(m))
	i := 0
	for k, v := range m {
		deployers[i] = &proto.DeployMetadata_Deployers{Type: k, Count: int32(v)}
		i++
	}
	return deployers
}

func updateOrAddKey(m map[proto.BuilderType]int, k proto.BuilderType) {
	if _, ok := m[k]; ok {
		m[k]++
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
