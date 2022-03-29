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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/util"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
)

type Kubectl struct {
	generate.Generator
	hydrationDir string
	labels       map[string]string

	transformAllowlist map[apimachinery.GroupKind]latestV2.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latestV2.ResourceFilter
}

func New(cfg render.Config, hydrationDir string, labels map[string]string) (Kubectl, error) {
	generator := generate.NewGenerator(cfg.GetWorkingDir(), cfg.GetRenderConfig().Generate, hydrationDir)
	transformAllowlist, transformDenylist, err := util.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return Kubectl{}, err
	}
	return Kubectl{
		Generator: generator,
		hydrationDir: hydrationDir,
		labels: labels,

		transformAllowlist: transformAllowlist,
		transformDenylist: transformDenylist,
	}, nil
}

func (r Kubectl) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool, _ string) error {
	return util.GenerateHydratedManifests(ctx, out, builds, r.Generator, r.hydrationDir, r.labels, r.transformAllowlist, r.transformDenylist)
}
