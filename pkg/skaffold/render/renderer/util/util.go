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

package util

import (
	"context"
	"io"
	"os"
	"strings"

	apim "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

type GenerateHydratedManifestsOptions struct {
	TransformAllowList         map[apim.GroupKind]latest.ResourceFilter
	TransformDenylist          map[apim.GroupKind]latest.ResourceFilter
	EnablePlatformNodeAffinity bool
	Offline                    bool
}

func GenerateHydratedManifests(ctx context.Context, out io.Writer, builds []graph.Artifact, g generate.Generator, labels map[string]string, ns string, opts GenerateHydratedManifestsOptions) (manifest.ManifestList, error) {
	// Generate manifests.
	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_generateManifest")
	manifests, err := g.Generate(rCtx, out)
	if err != nil {
		return nil, err
	}
	endTrace()

	// Update image labels.renderer_test.go
	rCtx, endTrace = instrumentation.StartTrace(ctx, "Render_setSkaffoldLabels")
	// TODO(aaron-prindle) wire proper transform allow/deny list args when going to V2
	manifests, err = manifests.ReplaceImages(rCtx, builds, manifest.NewResourceSelectorImages(opts.TransformAllowList, opts.TransformDenylist))
	if err != nil {
		return nil, err
	}
	rs := manifest.NewResourceSelectorLabels(opts.TransformAllowList, opts.TransformDenylist)
	// TODO(aaron-prindle) wire proper transform allow/deny list args when going to V2
	if manifests, err = manifests.SetLabels(labels, manifest.NewResourceSelectorLabels(opts.TransformAllowList, opts.TransformDenylist)); err != nil {
		return nil, err
	}
	// TODO(tejaldesai) consult with cloud deploy team if namespaces can be set in offline mode
	// in case namespace is set on the skaffold render cli command.
	if !opts.Offline {
		if manifests, err = manifests.SetNamespace(ns, rs); err != nil {
			return nil, err
		}
		endTrace()
	}

	if opts.Offline || !opts.EnablePlatformNodeAffinity {
		// TODO (gaghosh): To support platform node affinity in offline mode, we'll need to save the image platform
		// information in the build output file, and consume that here instead of looking up in the container registry.
		return manifests, nil
	}
	rCtx, endTrace = instrumentation.StartTrace(ctx, "Render_setPlatformNodeAffinity")
	platforms, err := manifests.GetImagePlatforms(rCtx, manifest.NewResourceSelectorImages(opts.TransformAllowList, opts.TransformDenylist))
	if err != nil {
		return nil, err
	}
	if manifests, err = manifests.SetPlatformNodeAffinity(rCtx, manifest.NewResourceSelectorAffinity(opts.TransformAllowList, opts.TransformDenylist), platforms); err != nil {
		return nil, err
	}
	endTrace()
	return manifests, nil
}

func ConsolidateTransformConfiguration(cfg render.Config) (map[apim.GroupKind]latest.ResourceFilter, map[apim.GroupKind]latest.ResourceFilter, error) {
	// TODO(aaron-prindle) currently this also modifies the flag & config to support a JSON path syntax for input.
	// this should be done elsewhere eventually

	transformableAllowlist := map[apim.GroupKind]latest.ResourceFilter{}
	transformableDenylist := map[apim.GroupKind]latest.ResourceFilter{}
	// add default values
	for _, rf := range manifest.TransformAllowlist {
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableAllowlist[groupKind] = ConvertJSONPathIndex(rf)
	}
	for _, rf := range manifest.TransformDenylist {
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableDenylist[groupKind] = ConvertJSONPathIndex(rf)
	}

	// add user schema values, override defaults
	for _, rf := range cfg.TransformAllowList() {
		instrumentation.AddResourceFilter("schema", "allow")
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableAllowlist[groupKind] = ConvertJSONPathIndex(rf)
		delete(transformableDenylist, groupKind)
	}
	for _, rf := range cfg.TransformDenyList() {
		instrumentation.AddResourceFilter("schema", "deny")
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableDenylist[groupKind] = ConvertJSONPathIndex(rf)
		delete(transformableAllowlist, groupKind)
	}

	// add user flag values, override user schema values and defaults
	// TODO(aaron-prindle) see if workdir needs to be considered in this read
	if cfg.TransformRulesFile() != "" {
		transformRulesFromFile, err := os.ReadFile(cfg.TransformRulesFile())
		if err != nil {
			return nil, nil, err
		}
		rsc := latest.ResourceSelectorConfig{}
		err = yaml.Unmarshal(transformRulesFromFile, &rsc)
		if err != nil {
			return nil, nil, err
		}
		for _, rf := range rsc.Allow {
			instrumentation.AddResourceFilter("cli-flag", "allow")
			groupKind := apim.ParseGroupKind(rf.GroupKind)
			transformableAllowlist[groupKind] = ConvertJSONPathIndex(rf)
			delete(transformableDenylist, groupKind)
		}

		for _, rf := range rsc.Deny {
			instrumentation.AddResourceFilter("cli-flag", "deny")
			groupKind := apim.ParseGroupKind(rf.GroupKind)
			transformableDenylist[groupKind] = ConvertJSONPathIndex(rf)
			delete(transformableAllowlist, groupKind)
		}
	}

	return transformableAllowlist, transformableDenylist, nil
}

func ConvertJSONPathIndex(rf latest.ResourceFilter) latest.ResourceFilter {
	nrf := latest.ResourceFilter{}
	nrf.GroupKind = rf.GroupKind
	if len(rf.Affinity) > 0 {
		naffinity := []string{}
		for _, str := range rf.Affinity {
			if str == ".*" {
				naffinity = append(naffinity, str)
				continue
			}
			nstr := strings.ReplaceAll(str, ".*", "")
			naffinity = append(naffinity, nstr)
		}
		nrf.Affinity = naffinity
	}

	if len(rf.Labels) > 0 {
		nlabels := []string{}
		for _, str := range rf.Labels {
			if str == ".*" {
				nlabels = append(nlabels, str)
				continue
			}
			nstr := strings.ReplaceAll(str, ".*", "")
			nlabels = append(nlabels, nstr)
		}
		nrf.Labels = nlabels
	}

	if len(rf.Image) > 0 {
		nimage := []string{}
		for _, str := range rf.Image {
			if str == ".*" {
				nimage = append(nimage, str)
				continue
			}
			nstr := strings.ReplaceAll(str, ".*", "")
			nimage = append(nimage, nstr)
		}
		nrf.Image = nimage
	}

	return nrf
}
