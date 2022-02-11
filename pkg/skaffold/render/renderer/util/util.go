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

package util

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
)

const (
	DryFileName = "manifests.yaml"
)

func GenerateHydratedManifests(ctx context.Context, out io.Writer, builds []graph.Artifact, g generate.Generator, hydrationDir string, labels map[string]string) error {
	// Generate manifests.
	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_generateManifest")
	if err := os.MkdirAll(hydrationDir, os.ModePerm); err != nil {
		return err
	}
	manifests, err := g.Generate(rCtx, out)
	if err != nil {
		return err
	}
	endTrace()

	// Update image labels.renderer_test.go
	rCtx, endTrace = instrumentation.StartTrace(ctx, "Render_setSkaffoldLabels")
	manifests, err = manifests.ReplaceImages(rCtx, builds)
	if err != nil {
		return err
	}
	if manifests, err = manifests.SetLabels(labels); err != nil {
		return err
	}
	endTrace()

	// Cache the dry manifests to the hydration directory.
	_, endTrace = instrumentation.StartTrace(ctx, "Render_cacheDryConfig")
	dryConfigPath := filepath.Join(hydrationDir, DryFileName)
	if err := manifest.Write(manifests.String(), dryConfigPath, out); err != nil {
		return err
	}
	endTrace()
	return nil
}
