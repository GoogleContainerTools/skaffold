/*
Copyright 2022 The Skaffold Authors

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

package noop

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// Noop renderer does nothing for the render phase.
// This struct is currently only used in conjunction with the Helm deployer.
type Noop struct{}

func New(_ latest.RenderConfig, _, _ string, _ map[string]string) Noop {
	return Noop{}
}

func (r Noop) Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	manifestListByConfig := manifest.NewManifestListByConfig()
	return manifestListByConfig, nil
}

func (r Noop) ManifestDeps() ([]string, error) {
	return nil, nil
}
