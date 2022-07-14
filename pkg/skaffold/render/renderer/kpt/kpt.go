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
	"os"
	"os/exec"
	"path/filepath"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	rUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/validate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type Kpt struct {
	configName string

	generate.Generator
	validate.Validator
	transform.Transformer
	hydrationDir string
	labels       map[string]string

	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

const (
	DryFileName = "manifests.yaml"
)

func New(cfg render.Config, rCfg latest.RenderConfig, hydrationDir string, labels map[string]string, configName string) (*Kpt, error) {
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate)
	transformAllowlist, transformDenylist, err := rUtil.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return nil, err
	}
	var validator validate.Validator
	if rCfg.Validate != nil {
		var err error
		validator, err = validate.NewValidator(*rCfg.Validate)
		if err != nil {
			return nil, err
		}
	} else {
		validator, _ = validate.NewValidator([]latest.Validator{})
	}

	var transformer transform.Transformer
	if rCfg.Transform != nil {
		var err error
		transformer, err = transform.NewTransformer(*rCfg.Transform)
		if err != nil {
			return nil, err
		}
	} else {
		transformer, _ = transform.NewTransformer([]latest.Transformer{})
	}
	return &Kpt{
		configName:         configName,
		Generator:          generator,
		Validator:          validator,
		Transformer:        transformer,
		hydrationDir:       hydrationDir,
		labels:             labels,
		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (r *Kpt) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool) (*manifest.ManifestListByConfig, error) {
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
			return nil, err
		}
		cmd := exec.CommandContext(ctx, "kpt", "pkg", "init", r.hydrationDir)
		if _, err := util.RunCmdOut(ctx, cmd); err != nil {
			return nil, sErrors.NewError(err,
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
	kptfileBytes, err := os.ReadFile(kptfilePath)
	if err != nil {
		endTrace(instrumentation.TraceEndError(fmt.Errorf("read Kptfile from %v: %w",
			filepath.Dir(kptfilePath), err)))
		return nil, err
	}
	if err := yaml.UnmarshalStrict(kptfileBytes, &kfConfig); err != nil {
		return nil, errors.ParseKptfileError(err, r.hydrationDir)
	}
	if err := os.RemoveAll(r.hydrationDir); err != nil {
		return nil, errors.DeleteKptfileError(err, r.hydrationDir)
	}
	endTrace()
	manifests, errH := rUtil.GenerateHydratedManifests(ctx, out, builds, r.Generator, r.labels, r.transformAllowlist, r.transformDenylist)
	if err != nil {
		return nil, errH
	}
	// Write generated dry manifests.
	_, endTrace = instrumentation.StartTrace(ctx, "Render_cacheDryConfig")
	dryConfigPath := filepath.Join(r.hydrationDir, DryFileName)
	if err := os.MkdirAll(r.hydrationDir, os.ModePerm); err != nil {
		return nil, err
	}
	if err := manifest.Write(manifests.String(), dryConfigPath, io.Discard); err != nil {
		return nil, err
	}
	endTrace()
	if kfConfig.Pipeline == nil {
		kfConfig.Pipeline = &kptfile.Pipeline{}
	}
	kfConfig.Pipeline.Validators = r.GetDeclarativeValidators()
	kfConfig.Pipeline.Mutators, err = r.GetDeclarativeTransformers()
	if err != nil {
		return nil, err
	}
	configByte, err := yaml.Marshal(kfConfig)
	if err != nil {
		return nil, err
	}
	if err = os.WriteFile(kptfilePath, configByte, 0644); err != nil {
		manifestListByConfig := manifest.NewManifestListByConfig()
		manifestListByConfig.Add(r.configName, manifests)
		return &manifestListByConfig, err
	}
	endTrace()

	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_kptRenderCommand")
	cmd := exec.CommandContext(rCtx, "kpt", "fn", "render", r.hydrationDir)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(ctx, cmd); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		// TODO(yuwenma): How to guide users when they face kpt error (may due to bad user config)?
		return nil, err
	}
	return r.unwrapManifests(ctx, out)
}

// unwrapManifests converts the structured manifest to a flatten format
func (r *Kpt) unwrapManifests(ctx context.Context, out io.Writer) (*manifest.ManifestListByConfig, error) {
	var m manifest.ManifestList
	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_outputManifests")
	cmd := exec.CommandContext(rCtx, "kpt", "fn", "source", r.hydrationDir, "-o", "unwrap")
	var buf []byte
	cmd.Stderr = out
	buf, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		manifestListByConfig := manifest.NewManifestListByConfig()
		manifestListByConfig.Add(r.configName, m)
		return &manifestListByConfig, err
	}
	m = append(m, buf)
	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(r.configName, m)
	return &manifestListByConfig, nil
}
