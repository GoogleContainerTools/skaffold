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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/validate"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	DefaultHydrationDir = ".kpt-pipeline"
	dryFileName         = "manifests.yaml"
)

type Renderer interface {
	Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool, outputPath string) error
}

// NewSkaffoldRenderer creates a new Renderer object from the latestV2 API schema.
func NewSkaffoldRenderer(config *latestV2.RenderConfig, workingDir string) (Renderer, error) {
	// TODO(yuwenma): return instance of kpt-managed mode or skaffold-managed mode defer to the config.Path fields.
	// The alpha implementation only has skaffold-managed mode.
	var hydrationDir string
	if config.Output == "" {
		hydrationDir = filepath.Join(workingDir, DefaultHydrationDir)
	} else {
		hydrationDir = config.Output
	}

	generator := generate.NewGenerator(workingDir, config.Generate, hydrationDir)
	var validator *validate.Validator
	if config.Validate != nil {
		var err error
		validator, err = validate.NewValidator(*config.Validate)
		if err != nil {
			return nil, err
		}
	} else {
		validator, _ = validate.NewValidator([]latestV2.Validator{})
	}

	var transformer *transform.Transformer
	if config.Transform != nil {
		var err error
		transformer, err = transform.NewTransformer(*config.Transform)
		if err != nil {
			return nil, err
		}
	} else {
		transformer, _ = transform.NewTransformer([]latestV2.Transformer{})
	}
	return &SkaffoldRenderer{Generator: *generator, Validator: *validator, Transformer: *transformer,
		workingDir: workingDir, hydrationDir: hydrationDir}, nil
}

type SkaffoldRenderer struct {
	generate.Generator
	validate.Validator
	transform.Transformer
	workingDir   string
	hydrationDir string
	labels       map[string]string
}

func (r *SkaffoldRenderer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool, _ string) error {
	kptfilePath := filepath.Join(r.hydrationDir, kptfile.KptFileName)
	kfConfig := &kptfile.KptFile{}

	// Initialize the kpt hydration directory.
	// This directory is used to cache DRY config and hydrates the DRY config to WET config in-place.
	// This is needed because kpt v1 only supports in-place config while users may not want to have their config be
	// hydrated in place.
	// Once Kptfile is initialized, its "pipeline" field will be updated in each skaffold render, and its "inventory"
	// will keep the same to guarantee accurate `kpt live apply`.
	_, endTrace := instrumentation.StartTrace(ctx, "Render_initKptfile")
	if _, err := os.Stat(r.hydrationDir); os.IsNotExist(err) {
		if err := os.MkdirAll(r.hydrationDir, os.ModePerm); err != nil {
			endTrace(instrumentation.TraceEndError(fmt.Errorf("create hydration dir %v:%w", r.hydrationDir, err)))
			return err
		}
		cmd := exec.CommandContext(ctx, "kpt", "pkg", "init", r.hydrationDir)
		if err := util.RunCmd(cmd); err != nil {
			endTrace(instrumentation.TraceEndError(fmt.Errorf("`kpt pkg init %v`:%w", r.hydrationDir, err)))
			return err
		}
	}
	endTrace()
	_, endTrace = instrumentation.StartTrace(ctx, "Render_readKptfile")
	file, err := os.Open(kptfilePath)
	if err != nil {
		endTrace(instrumentation.TraceEndError(fmt.Errorf("read Kptfile from %v: %w",
			filepath.Dir(kptfilePath), err)))
		return err
	}
	defer func() { _ = file.Close() }()
	if err := yaml.NewDecoder(file).Decode(&kfConfig); err != nil {
		return errors.ParseKptfileError(err, r.hydrationDir)
	}
	if err := os.RemoveAll(r.hydrationDir); err != nil {
		return errors.DeleteKptfileError(err, r.hydrationDir)
	}
	if err := os.MkdirAll(r.hydrationDir, os.ModePerm); err != nil {
		return err
	}
	endTrace()

	// Generate manifests.
	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_generateManifest")
	manifests, err := r.Generate(rCtx, out)
	if err != nil {
		return err
	}
	endTrace()

	// Update image labels.
	rCtx, endTrace = instrumentation.StartTrace(ctx, "Render_setSkaffoldLabels")
	manifests, err = manifests.ReplaceImages(rCtx, builds)
	if err != nil {
		return err
	}
	if manifests, err = manifests.SetLabels(r.labels); err != nil {
		return err
	}
	endTrace()

	// Cache the dry manifests to the hydration directory.
	_, endTrace = instrumentation.StartTrace(ctx, "Render_cacheDryConfig")
	dryConfigPath := filepath.Join(r.hydrationDir, dryFileName)
	if err := manifest.Write(manifests.String(), dryConfigPath, out); err != nil {
		return err
	}
	endTrace()

	// Refresh the Kptfile.
	_, endTrace = instrumentation.StartTrace(ctx, "Render_refreshKptfile")
	if kfConfig.Pipeline == nil {
		kfConfig.Pipeline = &kptfile.Pipeline{}
	}
	kfConfig.Pipeline.Validators = r.GetDeclarativeValidators()
	kfConfig.Pipeline.Mutators, err = r.GetDeclarativeTransformers()
	if err != nil {
		return err
	}
	configByte, err := yaml.Marshal(kfConfig)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(kptfilePath, configByte, 0644); err != nil {
		return err
	}
	endTrace()

	rCtx, endTrace = instrumentation.StartTrace(ctx, "Render_kptRenderCommand")
	cmd := exec.CommandContext(rCtx, "kpt", "fn", "render", r.hydrationDir)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		// TODO(yuwenma): How to guide users when they face kpt error (may due to bad user config)?
		return err
	}
	return nil
}
