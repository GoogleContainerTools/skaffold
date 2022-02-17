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

package kpt

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	rUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/validate"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type Kpt struct {
	generate.Generator
	validate.Validator
	transform.Transformer
	hydrationDir string
	labels       map[string]string
}

func New(config *latestV2.RenderConfig, workingDir, hydrationDir string,
	labels map[string]string) (*Kpt, error) {
	generator := generate.NewGenerator(workingDir, config.Generate, hydrationDir)
	var validator validate.Validator
	if config.Validate != nil {
		var err error
		validator, err = validate.NewValidator(*config.Validate)
		if err != nil {
			return nil, err
		}
	} else {
		validator, _ = validate.NewValidator([]latestV2.Validator{})
	}

	var transformer transform.Transformer
	if config.Transform != nil {
		var err error
		transformer, err = transform.NewTransformer(*config.Transform)
		if err != nil {
			return nil, err
		}
	} else {
		transformer, _ = transform.NewTransformer([]latestV2.Transformer{})
	}
	return &Kpt{Generator: generator, Validator: validator, Transformer: transformer,
		hydrationDir: hydrationDir, labels: labels}, nil
}

func (r *Kpt) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool, output string) error {

	kptfilePath := filepath.Join(r.hydrationDir, kptfile.KptFileName)
	kfConfig := &kptfile.KptFile{}

	// Initialize the kpt hydration directory.
	// This directory is used to cache DRY config and hydrates the DRY config to WET config in-place.
	// This is needed because kpt v1 only supports in-place config while users may not want to have their config be
	// hydrated in place.
	// Once Kptfile is initialized, its "pipeline" field will be updated in each skaffold render, and its "inventory"
	// will keep the same to guarantee accurate `kpt live apply`.
	_, endTrace := instrumentation.StartTrace(ctx, "Render_initKptfile")
	if _, err := os.Stat(kptfilePath); os.IsNotExist(err) {
		if err := os.MkdirAll(r.hydrationDir, os.ModePerm); err != nil {
			endTrace(instrumentation.TraceEndError(fmt.Errorf("create hydration dir %v:%w", r.hydrationDir, err)))
			return err
		}
		cmd := exec.CommandContext(ctx, "kpt", "pkg", "init", r.hydrationDir)
		if _, err := util.RunCmdOut(ctx, cmd); err != nil {
			return sErrors.NewError(err,
				&proto.ActionableErr{
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
	endTrace()
	_, endTrace = instrumentation.StartTrace(ctx, "Render_readKptfile")
	kptfileBytes, err := ioutil.ReadFile(kptfilePath)
	if err != nil {
		endTrace(instrumentation.TraceEndError(fmt.Errorf("read Kptfile from %v: %w",
			filepath.Dir(kptfilePath), err)))
		return err
	}
	if err := yaml.UnmarshalStrict(kptfileBytes, &kfConfig); err != nil {
		return errors.ParseKptfileError(err, r.hydrationDir)
	}
	if err := os.RemoveAll(r.hydrationDir); err != nil {
		return errors.DeleteKptfileError(err, r.hydrationDir)
	}
	endTrace()
	rUtil.GenerateHydratedManifests(ctx, out, builds, r.Generator, r.hydrationDir, r.labels)

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

	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_kptRenderCommand")
	cmd := exec.CommandContext(rCtx, "kpt", "fn", "render", r.hydrationDir)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(ctx, cmd); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		// TODO(yuwenma): How to guide users when they face kpt error (may due to bad user config)?
		return err
	}

	if output != "" {
		r.writeManifestsToFile(ctx, out, output)
	}
	return nil
}

// writeManifestsToFile converts the structured manifest to a flatten format and store them in the given `output` file.
func (r *Kpt) writeManifestsToFile(ctx context.Context, out io.Writer, output string) error {
	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_outputManifests")
	cmd := exec.CommandContext(rCtx, "kpt", "fn", "source", r.hydrationDir, "-o", "unwrap")
	var buf []byte
	cmd.Stderr = out
	buf, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	return ioutil.WriteFile(output, buf, os.ModePerm)
}
