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

package renderer

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/kubectl"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

type Renderer interface {
	Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool, output string) error
	// ManifestDeps returns the user kubenertes manifests to file watcher. In dev mode, a "redeploy" will be triggered
	// if any of the "Dependencies" manifest is changed.
	ManifestDeps() ([]string, error)
}

// New creates a new Renderer object from the latestV2 API schema.
func New(config *latestV2.RenderConfig, workingDir, hydrationDir string,
	labels map[string]string) (Renderer, error) {
	if config.Validate == nil && config.Transform == nil && config.Kpt == nil {
		return kubectl.New(config, workingDir, hydrationDir, labels)
	}
	return kpt.New(config, workingDir, hydrationDir, labels)
}
