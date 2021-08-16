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

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/validate"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	DefaultHydrationDir = ".kpt-pipeline"
	dryFileName         = "manifests.yaml"
)

type Renderer interface {
	Render(context.Context, io.Writer, []graph.Artifact) error
}

// NewSkaffoldRenderer creates a new Renderer object from the latestV2 API schema.
func NewSkaffoldRenderer(config *latestV2.RenderConfig, workingDir string) (Renderer, error) {
	// TODO(yuwenma): return instance of kpt-managed mode or skaffold-managed mode defer to the config.Path fields.
	// The alpha implementation only has skaffold-managed mode.
	// TODO(yuwenma): The current work directory may not be accurate if users use --filepath flag.
	hydrationDir := filepath.Join(workingDir, DefaultHydrationDir)

	generator := generate.NewGenerator(workingDir, config.Generate)
	/* TODO(yuwenma): Apply new UX
		if config.Generate == nil {
		// If render.generate is not given, default to current working directory.
		defaultManifests := filepath.Join(workingDir, "*.yaml")
		generator = generate.NewGenerator(workingDir, latestV2.Generate{Manifests: []string{defaultManifests}})
	} else {
		generator = generate.NewGenerator(workingDir, *config.Generate)
	}
	*/
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

// prepareHydrationDir guarantees the existence of a kpt-initialized temporary directory.
// This directory is used to cache DRY config and hydrates the DRY config to WET config in-place.
// This is needed because kpt v1 only supports in-place config while users may not want to have their config be
// hydrated in place.
func (r *SkaffoldRenderer) prepareHydrationDir(ctx context.Context) error {
	if _, err := os.Stat(r.hydrationDir); os.IsNotExist(err) {
		log.Entry(ctx).Debugf("creating render directory: %v", r.hydrationDir)
		if err := os.MkdirAll(r.hydrationDir, os.ModePerm); err != nil {
			return fmt.Errorf("creating render directory for hydration: %w", err)
		}
	}
	kptFilePath := filepath.Join(r.hydrationDir, kptfile.KptFileName)
	if _, err := os.Stat(kptFilePath); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "kpt", "pkg", "init", r.hydrationDir)
		if _, err := util.RunCmdOut(cmd); err != nil {
			return sErrors.NewError(err,
				proto.ActionableErr{
					Message: fmt.Sprintf("unable to initialize Kptfile in %v", r.hydrationDir),
					ErrCode: proto.StatusCode_RENDER_KPTFILE_INIT_ERR,
					Suggestions: []*proto.Suggestion{
						{
							SuggestionCode: proto.SuggestionCode_KPTFILE_MANUAL_INIT,
							Action:         fmt.Sprintf("please manually run `kpt pkg init %v`", r.hydrationDir),
						},
					},
				})
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
		return sErrors.NewError(err,
			proto.ActionableErr{
				Message: fmt.Sprintf("unable to parse Kptfile in %v", r.hydrationDir),
				ErrCode: proto.StatusCode_RENDER_KPTFILE_INVALID_YAML_ERR,
				Suggestions: []*proto.Suggestion{
					{
						SuggestionCode: proto.SuggestionCode_KPTFILE_CHECK_YAML,
						Action: fmt.Sprintf("please check if the Kptfile is correct and " +
							"the `apiVersion` is greater than `v1alpha2`"),
					},
				},
			})
	}
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
		return fmt.Errorf("unable to marshal Kptfile config %v", kfConfig)
	}
	if err = ioutil.WriteFile(kptFilePath, configByte, 0644); err != nil {
		return fmt.Errorf("unable to update %v", kptFilePath)
	}
	return nil
}
