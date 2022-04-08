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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	apim "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

const (
	DryFileName = "manifests.yaml"
)

func GenerateHydratedManifests(ctx context.Context, out io.Writer, builds []graph.Artifact, g generate.Generator, hydrationDir string, labels map[string]string, transformAllowlist, transformDenylist map[apim.GroupKind]latestV2.ResourceFilter) error {
	// Generate manifests.
	rCtx, endTrace := instrumentation.StartTrace(ctx, "Render_generateManifest")
	if err := os.MkdirAll(hydrationDir, os.ModePerm); err != nil {
		return err
	}
	manifests, err := g.Generate(rCtx, out)
	if err != nil {
		return err
	}
	endTrace()

	// Update image labels.renderer_test.go
	rCtx, endTrace = instrumentation.StartTrace(ctx, "Render_setSkaffoldLabels")
	// TODO(aaron-prindle) wire proper transform allow/deny list args when going to V2
	manifests, err = manifests.ReplaceImages(rCtx, builds, manifest.NewResourceSelectorImages(transformAllowlist, transformDenylist))
	if err != nil {
		return err
	}
	// TODO(aaron-prindle) wire proper transform allow/deny list args when going to V2
	if manifests, err = manifests.SetLabels(labels, manifest.NewResourceSelectorLabels(transformAllowlist, transformDenylist)); err != nil {
		return err
	}
	endTrace()

	// Cache the dry manifests to the hydration directory.
	_, endTrace = instrumentation.StartTrace(ctx, "Render_cacheDryConfig")
	dryConfigPath := filepath.Join(hydrationDir, DryFileName)
	if err := manifest.Write(manifests.String(), dryConfigPath, out); err != nil {
		return err
	}
	endTrace()
	return nil
}

func ConsolidateTransformConfiguration(cfg render.Config) (map[apim.GroupKind]latestV2.ResourceFilter, map[apim.GroupKind]latestV2.ResourceFilter, error) {
	// TODO(aaron-prindle) currently this also modifies the flag & config to support a JSON path syntax for input.
	// this should be done elsewhere eventually

	transformableAllowlist := map[apim.GroupKind]latestV2.ResourceFilter{}
	transformableDenylist := map[apim.GroupKind]latestV2.ResourceFilter{}
	// add default values
	for _, rf := range manifest.TransformAllowlist {
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableAllowlist[groupKind] = convertJSONPathIndex(rf)
	}
	for _, rf := range manifest.TransformDenylist {
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableDenylist[groupKind] = convertJSONPathIndex(rf)
	}

	// add user schema values, override defaults
	for _, rf := range cfg.TransformAllowList() {
		instrumentation.AddResourceFilter("schema", "allow")
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableAllowlist[groupKind] = convertJSONPathIndex(rf)
		delete(transformableDenylist, groupKind)
	}
	for _, rf := range cfg.TransformDenyList() {
		instrumentation.AddResourceFilter("schema", "deny")
		groupKind := apim.ParseGroupKind(rf.GroupKind)
		transformableDenylist[groupKind] = convertJSONPathIndex(rf)
		delete(transformableAllowlist, groupKind)
	}

	// add user flag values, override user schema values and defaults
	// TODO(aaron-prindle) see if workdir needs to be considered in this read
	if cfg.TransformRulesFile() != "" {
		transformRulesFromFile, err := ioutil.ReadFile(cfg.TransformRulesFile())
		if err != nil {
			return nil, nil, err
		}
		rsc := latestV2.ResourceSelectorConfig{}
		err = yaml.Unmarshal(transformRulesFromFile, &rsc)
		if err != nil {
			return nil, nil, err
		}
		for _, rf := range rsc.Allow {
			instrumentation.AddResourceFilter("cli-flag", "allow")
			groupKind := apim.ParseGroupKind(rf.GroupKind)
			transformableAllowlist[groupKind] = convertJSONPathIndex(rf)
			delete(transformableDenylist, groupKind)
		}

		for _, rf := range rsc.Deny {
			instrumentation.AddResourceFilter("cli-flag", "deny")
			groupKind := apim.ParseGroupKind(rf.GroupKind)
			transformableDenylist[groupKind] = convertJSONPathIndex(rf)
			delete(transformableAllowlist, groupKind)
		}
	}

	return transformableAllowlist, transformableDenylist, nil
}

func convertJSONPathIndex(rf latestV2.ResourceFilter) latestV2.ResourceFilter {
	nrf := latestV2.ResourceFilter{}
	nrf.GroupKind = rf.GroupKind

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
