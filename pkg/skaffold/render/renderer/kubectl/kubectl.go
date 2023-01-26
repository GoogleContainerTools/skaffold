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
	"bytes"
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"io"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	rUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type Kubectl struct {
	cfg                render.Config
	rCfg               latest.RenderConfig
	configName         string
	namespace          string
	generator          generate.Generator
	labels             map[string]string
	manifestOverrides  map[string]string
	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func New(cfg render.Config, rCfg latest.RenderConfig, labels map[string]string, configName string, ns string, manifestOverrides map[string]string) (Kubectl, error) {
	transformAllowlist, transformDenylist, err := rUtil.ConsolidateTransformConfiguration(cfg)
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate, "")
	if err != nil {
		return Kubectl{}, err
	}
	return Kubectl{
		cfg:                cfg,
		configName:         configName,
		rCfg:               rCfg,
		namespace:          ns,
		generator:          generator,
		labels:             labels,
		manifestOverrides:  manifestOverrides,
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
	manifests, err := r.generator.Generate(ctx, out)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}
	var tra []latest.Transformer
	if r.rCfg.Transform != nil {
		tra = *r.rCfg.Transform
	}
	mutators, err := transform.NewTransformer(tra)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}
	transformers, err := mutators.GetDeclarativeTransformers()
	if len(r.manifestOverrides) > 0 {
		transformers = append(transformers, kptfile.Function{Image: "gcr.io/kpt-fn/apply-setters:unstable", ConfigMap: r.manifestOverrides})
	}

	for _, transformer := range transformers {
		var kvs []string
		for key, value := range transformer.ConfigMap {
			kvs = append(kvs, fmt.Sprintf("%s=%s", key, value))
		}
		fmt.Println(kvs)

		args := []string{"fn", "eval", "-o", "unwrap", "-i", transformer.Image, "-", "--"}
		args = append(args, kvs...)
		command := exec.Command("kpt", args...)
		command.Stdin = manifests.Reader()
		output, err := command.Output()
		fmt.Println(string(output))
		if err != nil {
			fmt.Println(err.Error())
		}

		manifests, err = manifest.Load(bytes.NewBuffer(output))
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	opts := rUtil.GenerateHydratedManifestsOptions{
		TransformAllowList:         r.transformAllowlist,
		TransformDenylist:          r.transformDenylist,
		EnablePlatformNodeAffinity: r.cfg.EnablePlatformNodeAffinityInRenderedManifests(),
		EnableGKEARMNodeToleration: r.cfg.EnableGKEARMNodeTolerationInRenderedManifests(),
		Offline:                    offline,
		KubeContext:                r.cfg.GetKubeContext(),
	}
	manifests, err = rUtil.BaseTransform(ctx, manifests, builds, opts, r.labels, r.namespace)

	endTrace()
	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(r.configName, manifests)
	return manifestListByConfig, err
}

func (r Kubectl) ManifestDeps() ([]string, error) {
	return nil, nil
}
