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

package kubectl

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/util"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

type Kubectl struct {
	generate.Generator
	hydrationDir string
	labels       map[string]string
}

func New(config *latestV2.RenderConfig, workingDir, hydrationDir string,
	labels map[string]string) (Kubectl, error) {
	generator := generate.NewGenerator(workingDir, config.Generate, hydrationDir)
	return Kubectl{Generator: generator, hydrationDir: hydrationDir, labels: labels}, nil
}

func (r Kubectl) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool, _ string) error {
	return util.GenerateHydratedManifests(ctx, out, builds, r.Generator, r.hydrationDir, r.labels)
}
