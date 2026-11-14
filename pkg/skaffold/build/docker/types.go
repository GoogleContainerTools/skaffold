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

package docker

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// Builder is an artifact builder that uses docker
type Builder struct {
	localDocker        docker.LocalDaemon
	cfg                docker.Config
	pushImages         bool
	useCLI             bool
	useBuildKit        *bool
	buildx             bool
	artifacts          ArtifactResolver
	sourceDependencies TransitiveSourceDependenciesResolver
}

// ArtifactResolver provides an interface to resolve built artifact tags by image name.
type ArtifactResolver interface {
	GetImageTag(imageName string) (string, bool)
}

// TransitiveSourceDependenciesResolver provides an interface to to evaluate the source dependencies for artifacts.
type TransitiveSourceDependenciesResolver interface {
	TransitiveArtifactDependencies(ctx context.Context, a *latest.Artifact) ([]string, error)
}

// NewBuilder returns an new instance of a docker builder
func NewArtifactBuilder(localDocker docker.LocalDaemon, cfg docker.Config, useCLI bool, useBuildKit *bool, buildx bool, pushImages bool, ar ArtifactResolver, dr TransitiveSourceDependenciesResolver) *Builder {
	return &Builder{
		localDocker:        localDocker,
		pushImages:         pushImages,
		cfg:                cfg,
		useCLI:             useCLI,
		useBuildKit:        useBuildKit,
		buildx:             buildx,
		artifacts:          ar,
		sourceDependencies: dr,
	}
}
