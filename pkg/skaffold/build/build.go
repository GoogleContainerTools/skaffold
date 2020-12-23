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

package build

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Artifact is the result corresponding to each successful build.
type Artifact struct {
	ImageName string `json:"imageName"`
	Tag       string `json:"tag"`
}

// ArtifactGraph is a map of [artifact image : artifact definition]
type ArtifactGraph map[string]*latest.Artifact

// ToArtifactGraph creates an instance of `ArtifactGraph` from `[]*latest.Artifact`
func ToArtifactGraph(artifacts []*latest.Artifact) ArtifactGraph {
	m := make(map[string]*latest.Artifact)
	for _, a := range artifacts {
		m[a.ImageName] = a
	}
	return m
}

// Dependencies returns the de-referenced slice of required artifacts for a given artifact
func (g ArtifactGraph) Dependencies(a *latest.Artifact) []*latest.Artifact {
	var sl []*latest.Artifact
	for _, d := range a.Dependencies {
		sl = append(sl, g[d.ImageName])
	}
	return sl
}

// Builder is an interface to the Build API of Skaffold.
// It must build and make the resulting image accessible to the cluster.
// This could include pushing to a authorized repository or loading the nodes with the image.
// If artifacts is supplied, the builder should only rebuild those artifacts.
type Builder interface {
	Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]Artifact, error)

	// Prune removes images built with Skaffold
	Prune(context.Context, io.Writer) error
}

// PipelineBuilder is an interface for a specific Skaffold config pipeline build type.
// Current implementations are the `local`, `cluster` and `gcb`
type PipelineBuilder interface {

	// PreBuild executes any one-time setup required prior to starting any build on this builder
	PreBuild(ctx context.Context, out io.Writer) error

	// Build returns the `ArtifactBuilder` based on this build pipeline type
	Build(ctx context.Context, out io.Writer, artifact *latest.Artifact) ArtifactBuilder

	// PostBuild executes any one-time teardown required after all builds on this builder are complete
	PostBuild(ctx context.Context, out io.Writer) error

	// Concurrency specifies the max number of builds that can run at any one time. If concurrency is 0, then all builds can run in parallel.
	Concurrency() int

	// Prune removes images built in this pipeline
	Prune(context.Context, io.Writer) error
}

type ErrSyncMapNotSupported struct{}

func (ErrSyncMapNotSupported) Error() string {
	return "SyncMap is not supported by this builder"
}
