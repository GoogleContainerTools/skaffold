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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	DefaultHydrationDir = ".kpt-pipeline"
	dryFileName         = "manifests.yaml"
	KptFileName         = "Kptfile"
)

type Renderer interface {
	Render(context.Context, io.Writer, []graph.Artifact) error
}

func NewSkaffoldRenderer(config *latestV2.RenderConfig, workingDir string) Renderer {
	// TODO(yuwenma): return instance of kpt-managed mode or skaffold-managed mode defer to the config.Path fields.
	// The alpha implementation only has skaffold-managed mode.
	generator := generate.NewGenerator(workingDir, *config.Generate)
	return &SkaffoldRenderer{Generator: *generator, workingDir: workingDir}
}

type SkaffoldRenderer struct {
	generate.Generator
	workingDir string
	labels     map[string]string
}

// prepareHydrationDir guarantees the existence of a kpt-initialized temporary directory.
// This directory is used to cache DRY config and hydrates the DRY config to WET config in-place.
// This is needed because kpt v1 only supports in-place config while users may not want to have their config be
// hydrated in place.
func (r *SkaffoldRenderer) prepareHydrationDir(ctx context.Context) error {
	// TODO(yuwenma): The current work directory may not be accurate if users use --filepath flag.
	outputPath := filepath.Join(r.workingDir, DefaultHydrationDir)
	logrus.Debugf("creating render directory: %v", outputPath)

	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating cache directory for hydration: %w", err)
	}
	if _, err := os.Stat(filepath.Join(outputPath, KptFileName)); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "kpt", "pkg", "init", outputPath)
		if _, err := util.RunCmdOut(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (r *SkaffoldRenderer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	if err := r.prepareHydrationDir(ctx); err != nil {
		return err
	}

	manifests, err := r.Generate(ctx)
	if err != nil {
		return err
	}
	manifests, err = manifests.ReplaceImages(builds)
	if err != nil {
		return err
	}
	manifests.SetLabels(r.labels)

	// cache the dry manifests to the temp directory. manifests.yaml will be truncated if already exists.
	dryConfigPath := filepath.Join(r.workingDir, DefaultHydrationDir, dryFileName)
	if err := manifest.Write(manifests.String(), dryConfigPath, out); err != nil {
		return err
	}

	// TODO: mutate and validate
	return nil
}
