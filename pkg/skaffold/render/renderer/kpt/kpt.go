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
	"os/exec"

	"github.com/blang/semver"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

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

	if len(manifestOverrides) > 0 {
		err := transformer.Append(latest.Transformer{Name: "apply-setters", ConfigMap: util.EnvMapToSlice(manifestOverrides, ":")})
		if err != nil {
			return nil, err
		}
	}

	return &Kpt{
		cfg:                cfg,
		configName:         configName,
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
		cmd := exec.Command("kpt", "fn", "render", p, "-o", "unwrap")

		if buf, err := util.RunCmdOut(rCtx, cmd); err == nil {
			manifestList.Append(buf)
		} else {
			endTrace(instrumentation.TraceEndError(err))
			// TODO(yuwenma): How to guide users when they face kpt error (may due to bad user config)?
			return ml, err
		}
	}

	manifestList, err := r.Transformer.Transform(ctx, manifestList)

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	opts := rUtil.GenerateHydratedManifestsOptions{
		TransformAllowList:         r.transformAllowlist,
		TransformDenylist:          r.transformDenylist,
		EnablePlatformNodeAffinity: r.cfg.EnablePlatformNodeAffinityInRenderedManifests(),
		EnableGKEARMNodeToleration: r.cfg.EnableGKEARMNodeTolerationInRenderedManifests(),
		Offline:                    offline,
		KubeContext:                r.cfg.GetKubeContext(),
		InjectNamespace:            true,
	}

	manifestList, err = rUtil.BaseTransform(ctx, manifestList, builds, opts, r.labels, r.namespace)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	err = r.Validator.Validate(ctx, manifestList)

	if err != nil {
		return manifest.ManifestListByConfig{}, err
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
