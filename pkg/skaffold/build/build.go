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

// Result contains the result of a build of a single target artifact.
// It contains the target artifact (parsed from the skaffold config),
// and either an Artifact representing the result of a successful build,
// or the error encountered while executing the build.
type Result struct {
	Target latest.Artifact
	Result Artifact
	Error  error
}

// Builder is an interface to the Build API of Skaffold.
// It must build and make the resulting image accesible to the cluster.
// This could include pushing to a authorized repository or loading the nodes with the image.
// If artifacts is supplied, the builder should only rebuild those artifacts.
type Builder interface {
	Labels() map[string]string

	Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]chan Result, error)

	DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error)

	// Prune removes images built with Skaffold
	Prune(context.Context, io.Writer) error
}
