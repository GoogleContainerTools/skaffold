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
	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	DefaultHydrationDir = ".kpt-pipeline"
	dryFileName         = "manifests.yaml"
)

type Renderer interface {
	Render(context.Context, io.Writer, []graph.Artifact) error
}

// NewSkaffoldRenderer creates a new Renderer object from the latestV2 API schema.
func NewSkaffoldRenderer(config *latestV2.RenderConfig, workingDir string) Renderer {
	// TODO(yuwenma): return instance of kpt-managed mode or skaffold-managed mode defer to the config.Path fields.
	// The alpha implementation only has skaffold-managed mode.
	// TODO(yuwenma): The current work directory may not be accurate if users use --filepath flag.
	hydrationDir := filepath.Join(workingDir, DefaultHydrationDir)
	generator := generate.NewGenerator(workingDir, *config.Generate)
	return &SkaffoldRenderer{Generator: *generator, workingDir: workingDir, hydrationDir: hydrationDir}
}

type SkaffoldRenderer struct {
	generate.Generator
	workingDir   string
	hydrationDir string
	labels       map[string]string
}

// prepareHydrationDir guarantees the existence of a kpt-initialized temporary directory.
// This directory is used to cache DRY config and hydrates the DRY config to WET config in-place.
// This is needed because kpt v1 only supports in-place config while users may not want to have their config be
// hydrated in place.
func (r *SkaffoldRenderer) prepareHydrationDir(ctx context.Context) error {
	if _, err := os.Stat(r.hydrationDir); os.IsNotExist(err) {
		logrus.Debugf("creating render directory: %v", r.hydrationDir)
		if err := os.MkdirAll(r.hydrationDir, os.ModePerm); err != nil {
			return fmt.Errorf("creating cache directory for hydration: %w", err)
		}
	}
	kptFilePath := filepath.Join(r.hydrationDir, kptfile.KptFileName)
	if _, err := os.Stat(kptFilePath); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "kpt", "pkg", "init", r.hydrationDir)
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
	manifests, err = manifests.ReplaceImages(ctx, builds)
	if err != nil {
		return err
	}
	manifests.SetLabels(r.labels)

	// cache the dry manifests to the temp directory. manifests.yaml will be truncated if already exists.
	dryConfigPath := filepath.Join(r.hydrationDir, dryFileName)
	if err := manifest.Write(manifests.String(), dryConfigPath, out); err != nil {
		return err
	}

	// Read the existing Kptfile content. Kptfile is guaranteed to be exist in prepareHydrationDir.
	kptFilePath := filepath.Join(r.hydrationDir, kptfile.KptFileName)
	file, err := os.Open(kptFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	kfConfig := &kptfile.KptFile{}
	if err := yaml.NewDecoder(file).Decode(&kfConfig); err != nil {
		return err
	}

	// TODO: Update the Kptfile with the new validators.
	// TODO: Update the Kptfile with the new mutators.
	return nil
}
