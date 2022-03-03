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

package loader

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
)

// ImageLoader defines the behavior for loading images into a cluster.
type ImageLoader interface {
	// LoadImages loads the images into the cluster.
	LoadImages(context.Context, io.Writer, []graph.Artifact, []graph.Artifact, []graph.Artifact) error
}

type NoopImageLoader struct{}

func (n *NoopImageLoader) LoadImages(context.Context, io.Writer, []graph.Artifact, []graph.Artifact, []graph.Artifact) error {
	return nil
}

func (n *NoopImageLoader) TrackBuildArtifacts([]graph.Artifact, []graph.Artifact) {}
