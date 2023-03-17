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
	"io"
	"os/exec"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
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
	injectNs          bool
	pkgDir            []string
	manifestOverrides map[string]string

	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func New(cfg render.Config, rCfg latest.RenderConfig, hydrationDir string, labels map[string]string, configName string, ns string, manifestOverrides map[string]string, injectNs bool) (*Kpt, error) {
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
		injectNs:           injectNs,
	}, nil
}

func (r *Kpt) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	var ml manifest.ManifestListByConfig
	var manifestList manifest.ManifestList

	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_kptRenderCommand")

	for _, p := range r.pkgDir {
		cmd := exec.Command("kpt", "fn", "render", p, "-o", "unwrap")

		if buf, err := util.RunCmdOut(rCtx, cmd); err == nil {
			reader := kio.ByteReader{Reader: bytes.NewBuffer(buf)}
			b := bytes.NewBuffer([]byte{})
			writer := kio.ByteWriter{Writer: b}
			// Kpt fn render outputs Kptfile and Config data files content in result, we don't want them in our manifestList as these cannot be deployed to k8s cluster.
			pipeline := kio.Pipeline{Filters: []kio.Filter{framework.ResourceMatcherFunc(func(node *yaml.RNode) bool {
				meta, _ := node.GetMeta()
				return node.GetKind() != kptfile.KptFileKind && meta.Annotations["config.kubernetes.io/local-config"] != "true"
			})},
				Inputs:  []kio.Reader{&reader},
				Outputs: []kio.Writer{writer},
			}
			if err := pipeline.Execute(); err != nil {
				return ml, err
			}
			manifestList.Append(b.Bytes())
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
		InjectNamespace:            r.injectNs,
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
