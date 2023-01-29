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
	"bytes"
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/blang/semver"
	"io"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	rUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/validate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type Kpt struct {
	cfg        render.Config
	configName string

	generate.Generator
	validate.Validator
	transform.Transformer
	hydrationDir      string
	labels            map[string]string
	namespace         string
	pkgDir            []string
	manifestOverrides map[string]string

	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

const (
	DryFileName = "manifests.yaml"
)

var (
	KptVersion                      = currentKptVersion
	maxKptVersionAllowedForDeployer = "1.0.0-beta.24"
)

func New(cfg render.Config, rCfg latest.RenderConfig, hydrationDir string, labels map[string]string, configName string, ns string, manifestOverrides map[string]string) (*Kpt, error) {
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate, hydrationDir)
	transformAllowlist, transformDenylist, err := rUtil.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return nil, err
	}
	var validator validate.Validator
	if rCfg.Validate != nil {
		validator, err = validate.NewValidator(*rCfg.Validate)
		if err != nil {
			return nil, err
		}
	}

	var transformer transform.Transformer
	if rCfg.Transform != nil {
		transformer, err = transform.NewTransformer(*rCfg.Transform)
		if err != nil {
			return nil, err
		}
	}

	return &Kpt{
		cfg:                cfg,
		configName:         configName,
		Generator:          generator,
		pkgDir:             rCfg.Kpt,
		manifestOverrides:  manifestOverrides,
		Validator:          validator,
		Transformer:        transformer,
		hydrationDir:       hydrationDir,
		labels:             labels,
		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
		namespace:          ns,
	}, nil
}

func (r *Kpt) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	var ml manifest.ManifestListByConfig
	var manifestList manifest.ManifestList

	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_kptRenderCommand")

	for _, p := range r.pkgDir {
		cmd := exec.CommandContext(rCtx, "kpt", "fn", "render", p, "-o", "unwrap")
		buf := &bytes.Buffer{}
		cmd.Stdout = buf
		cmd.Stderr = out

		if err := util.RunCmd(ctx, cmd); err != nil {
			endTrace(instrumentation.TraceEndError(err))
			// TODO(yuwenma): How to guide users when they face kpt error (may due to bad user config)?
			return ml, err
		}
		manifestList.Append(buf.Bytes())
	}

	transformers, err := r.Transformer.GetDeclarativeTransformers()
	if err != nil {
		return ml, err
	}
	if len(r.manifestOverrides) > 0 {
		transformers = append(transformers, kptfile.Function{Image: "gcr.io/kpt-fn/apply-setters:unstable", ConfigMap: r.manifestOverrides})
	}
	for _, transformer := range transformers {
		slice := util.EnvMapToSlice(transformer.ConfigMap, "=")
		args := []string{"fn", "eval", "-i", transformer.Image, "-o", "unwrap", "-", "--"}
		args = append(args, slice...)
		cmd := exec.CommandContext(rCtx, "kpt", args...)
		reader := manifestList.Reader()
		buffer := &bytes.Buffer{}
		cmd.Stdin = reader
		cmd.Stdout = buffer

		fmt.Println(cmd.Args)
		err := cmd.Run()
		if err != nil {
			return ml, err
		}
		manifestList, err = manifest.Load(buffer)

	}

	opts := rUtil.GenerateHydratedManifestsOptions{
		TransformAllowList:         r.transformAllowlist,
		TransformDenylist:          r.transformDenylist,
		EnablePlatformNodeAffinity: r.cfg.EnablePlatformNodeAffinityInRenderedManifests(),
		EnableGKEARMNodeToleration: r.cfg.EnableGKEARMNodeTolerationInRenderedManifests(),
		Offline:                    offline,
		KubeContext:                r.cfg.GetKubeContext(),
	}

	manifestList, err = rUtil.BaseTransform(ctx, manifestList, builds, opts, r.labels, r.namespace)

	validators := r.Validator.GetDeclarativeValidators()

	if len(validators) > 0 {
		for _, validator := range validators {
			kvs := util.EnvMapToSlice(validator.ConfigMap, "=")
			args := []string{"fn", "eval", "-i", validator.Image, "-o", "unwrap", "-", "--"}
			args = append(args, kvs...)
			cmd := exec.CommandContext(rCtx, "kpt", args...)
			reader := manifestList.Reader()
			buffer := &bytes.Buffer{}
			cmd.Stdin = reader
			cmd.Stdout = buffer

			fmt.Println(cmd.Args)
			err := cmd.Run()
			if err != nil {
				return ml, err
			}
			manifestList, err = manifest.Load(buffer)
		}
	}

	ml = manifest.NewManifestListByConfig()
	ml.Add(r.configName, manifestList)
	return ml, err
}

func CheckIsProperBinVersion(ctx context.Context) error {
	maxAllowedVersion := semver.MustParse(maxKptVersionAllowedForDeployer)
	version, err := KptVersion(ctx)
	if err != nil {
		return err
	}

	currentVersion, err := semver.ParseTolerant(version)
	if err != nil {
		return err
	}

	if currentVersion.GT(maxAllowedVersion) {
		return fmt.Errorf("max allowed verion for Kpt renderer without Kpt deployer is %v, detected version is %v", maxKptVersionAllowedForDeployer, currentVersion)
	}

	return nil
}

func currentKptVersion(ctx context.Context) (string, error) {
	cmd := exec.Command("kpt", "version")
	b, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("kpt version command failed: %w", err)
	}
	version := string(b)
	return version, nil
}
