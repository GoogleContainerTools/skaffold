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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
)

// Builder is an interface to the Build API of Skaffold.
// It must build and make the resulting image accessible to the cluster.
// This could include pushing to a authorized repository or loading the nodes with the image.
// If artifacts is supplied, the builder should only rebuild those artifacts.
type Builder interface {
	Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latestV2.Artifact) ([]graph.Artifact, error)

	// Prune removes images built with Skaffold
	Prune(context.Context, io.Writer) error
}

// PipelineBuilder is an interface for a specific Skaffold config pipeline build type.
// Current implementations are the `local`, `cluster` and `gcb`
type PipelineBuilder interface {

	// PreBuild executes any one-time setup required prior to starting any build on this builder
	PreBuild(ctx context.Context, out io.Writer) error

	// Build returns the `ArtifactBuilder` based on this build pipeline type
	Build(ctx context.Context, out io.Writer, artifact *latestV2.Artifact) ArtifactBuilder

	// PostBuild executes any one-time teardown required after all builds on this builder are complete
	PostBuild(ctx context.Context, out io.Writer) error

	// Concurrency specifies the max number of builds that can run at any one time. If concurrency is 0, then all builds can run in parallel.
	Concurrency() int

	// Prune removes images built in this pipeline
	Prune(context.Context, io.Writer) error

	// PushImages specifies if the built image needs to be explicitly pushed to an image registry.
	PushImages() bool
}

type ErrSyncMapNotSupported struct{}

func (ErrSyncMapNotSupported) Error() string {
	return "SyncMap is not supported by this builder"
}

type ErrCustomBuildNoDockerfile struct{}

func (ErrCustomBuildNoDockerfile) Error() string {
	return "inferred sync with custom build requires explicitly declared Dockerfile dependency"
}
