/*
Copyright 2018 The Skaffold Authors

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

package build

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
)

// DependenciesForArtifact is used in tests.
var DependenciesForArtifact = dependenciesForArtifact

func dependenciesForArtifact(a *v1alpha2.Artifact) ([]string, error) {
	switch {
	case a.DockerArtifact != nil:
		return docker.GetDependencies(a.Workspace, a.DockerArtifact.DockerfilePath)

	case a.BazelArtifact != nil:
		return bazel.GetDependencies(a)

	default:
		return nil, fmt.Errorf("undefined artifact type: %+v", a.ArtifactType)
	}
}
