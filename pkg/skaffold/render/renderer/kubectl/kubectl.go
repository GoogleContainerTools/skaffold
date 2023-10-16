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

package kubectl

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	applysetters "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/applysetters"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	rUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/validate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type Kubectl struct {
	cfg                render.Config
	rCfg               latest.RenderConfig
	configName         string
	namespace          string
	injectNs           bool
	Generator          generate.Generator
	labels             map[string]string
	manifestOverrides  map[string]string
	transformer        transform.Transformer
	applySetters       applysetters.ApplySetters
	validator          validate.Validator
	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func New(cfg render.Config, rCfg latest.RenderConfig, labels map[string]string, configName string, ns string, manifestOverrides map[string]string, injectNs bool) (Kubectl, error) {
	transformAllowlist, transformDenylist, err := rUtil.ConsolidateTransformConfiguration(cfg)
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate, "")
	if err != nil {
		return Kubectl{}, err
	}

	var validator validate.Validator
	if rCfg.Validate != nil {
		validator, err = validate.NewValidator(*rCfg.Validate)
		if err != nil {
			return Kubectl{}, err
		}
	}

	var transformer transform.Transformer
	if rCfg.Transform != nil {
		transformer, err = transform.NewTransformer(*rCfg.Transform)
		if err != nil {
			return Kubectl{}, err
		}
	}

	var ass applysetters.ApplySetters
	if len(manifestOverrides) > 0 {
		for k, v := range manifestOverrides {
			ass.Setters = append(ass.Setters, applysetters.Setter{Name: k, Value: v})
		}
	}

	return Kubectl{
		cfg:                cfg,
		configName:         configName,
		rCfg:               rCfg,
		namespace:          ns,
		injectNs:           injectNs,
		Generator:          generator,
		labels:             labels,
		manifestOverrides:  manifestOverrides,
		validator:          validator,
		transformer:        transformer,
		applySetters:       ass,
		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (r Kubectl) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "Render_KubectlManifests")
	log.Entry(ctx).Infof("starting render process")
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"RendererType": "kubectl",
	})
	// get manifest contents from rawManifests and remoteManifests
	manifests, err := r.Generator.Generate(ctx, out)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	manifests, err = r.transformer.Transform(ctx, manifests)

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	manifests, err = r.applySetters.Apply(ctx, manifests)
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
	manifests, err = rUtil.BaseTransform(ctx, manifests, builds, opts, r.labels, r.namespace)

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	err = r.validator.Validate(ctx, manifests)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	endTrace()
	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(r.configName, manifests)
	return manifestListByConfig, err
}

func (r Kubectl) ManifestDeps() ([]string, error) {
	var localPaths []string
	for _, path := range r.rCfg.RawK8s {
		switch {
		case util.IsURL(path):
		case strings.HasPrefix(path, "gs://"):
		default:
			localPaths = append(localPaths, path)
		}
	}

	dependencyPaths, err := util.ExpandPathsGlob(r.cfg.GetWorkingDir(), localPaths)
	if err != nil {
		return []string{}, err
	}
	var deps []string

	for _, path := range dependencyPaths {
		err := filepath.Walk(path,
			func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				fname := filepath.Base(p)
				if strings.HasSuffix(fname, ".yaml") || strings.HasSuffix(fname, ".yml") || fname == kptfile.KptFileName {
					deps = append(deps, p)
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}
	return deps, nil
}
