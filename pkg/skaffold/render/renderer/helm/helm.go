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

package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	sUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type Helm struct {
	configName string
	generate.Generator
	config *latest.Helm

	kubeContext       string
	kubeConfig        string
	namespace         string
	configFile        string
	labels            map[string]string
	enableDebug       bool
	overrideProtocols []string

	manifestOverrides  map[string]string
	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (h Helm) EnableDebug() bool           { return h.enableDebug }
func (h Helm) OverrideProtocols() []string { return h.overrideProtocols }
func (h Helm) ConfigFile() string          { return h.configFile }
func (h Helm) KubeContext() string         { return h.kubeContext }
func (h Helm) KubeConfig() string          { return h.kubeConfig }
func (h Helm) Labels() map[string]string   { return h.labels }
func (h Helm) GlobalFlags() []string       { return h.config.Flags.Global }

func (h Helm) ManifestOverrides() map[string]string {
	return h.manifestOverrides
}

func New(cfg render.Config, rCfg latest.RenderConfig, labels map[string]string, configName string, manifestOverrides map[string]string) (Helm, error) {
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate, "")
	transformAllowlist, transformDenylist, err := util.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return Helm{}, err
	}
	return Helm{
		configName: configName,
		Generator:  generator,
		config:     rCfg.Helm,

		enableDebug:       cfg.Mode() == config.RunModes.Debug,
		overrideProtocols: debug.Protocols,
		configFile:        cfg.ConfigurationFile(),
		kubeContext:       cfg.GetKubeContext(),
		kubeConfig:        cfg.GetKubeConfig(),
		labels:            labels,
		namespace:         cfg.GetKubeNamespace(),
		manifestOverrides: manifestOverrides,

		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (h Helm) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool) (manifest.ManifestListByConfig, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "Render_HelmManifests")
	log.Entry(ctx).Infof("rendering using helm")
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"RendererType": "helm",
	})

	manifests, err := h.generateHelmManifests(ctx, builds)
	endTrace()
	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(h.configName, manifests)
	return manifestListByConfig, err
}

func (h Helm) generateHelmManifests(ctx context.Context, builds []graph.Artifact) (manifest.ManifestList, error) {
	var renderedManifests manifest.ManifestList
	helmEnv := sUtil.OSEnviron()
	var postRendererArgs []string

	if len(builds) > 0 {
		skaffoldBinary, filterEnv, cleanup, err := helm.PrepareSkaffoldFilter(h, builds)
		if err != nil {
			return nil, fmt.Errorf("could not prepare `skaffold filter`: %w", err)
		}
		// need to include current environment, specifically for HOME to lookup ~/.kube/config
		helmEnv = append(helmEnv, filterEnv...)
		postRendererArgs = []string{"--post-renderer", skaffoldBinary}
		defer cleanup()
	}

	for _, release := range h.config.Releases {
		releaseName, err := sUtil.ExpandEnvTemplateOrFail(release.Name, nil)
		if err != nil {
			return nil, helm.UserErr(fmt.Sprintf("cannot expand release name %q", release.Name), err)
		}

		release.ChartPath, err = sUtil.ExpandEnvTemplateOrFail(release.ChartPath, nil)
		if err != nil {
			return nil, helm.UserErr(fmt.Sprintf("cannot expand chart path %q", release.ChartPath), err)
		}

		args := []string{"template", releaseName, helm.ChartSource(release)}
		args = append(args, postRendererArgs...)
		if release.Packaged == nil && release.Version != "" {
			args = append(args, "--version", release.Version)
		}

		args, err = helm.ConstructOverrideArgs(&release, builds, args, h.manifestOverrides)
		if err != nil {
			return nil, helm.UserErr("construct override args", err)
		}

		if release.SkipTests {
			args = append(args, "--skip-tests")
		}

		namespace, err := helm.ReleaseNamespace(h.namespace, release)
		if err != nil {
			return nil, err
		}
		if h.namespace != "" {
			namespace = h.namespace
		}
		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}

		if release.Repo != "" {
			args = append(args, "--repo")
			args = append(args, release.Repo)
		}

		outBuffer := new(bytes.Buffer)
		errBuffer := new(bytes.Buffer)

		// Build Chart dependencies, but allow a user to skip it.
		if !release.SkipBuildDependencies && release.ChartPath != "" {
			log.Entry(ctx).Info("Building helm dependencies...")
			if err := helm.ExecWithStdoutAndStderr(ctx, h, io.Discard, errBuffer, false, helmEnv, "dep", "build", release.ChartPath); err != nil {
				log.Entry(ctx).Infof(errBuffer.String())
				return nil, helm.UserErr("building helm dependencies", err)
			}
		}

		err = helm.ExecWithStdoutAndStderr(ctx, h, outBuffer, errBuffer, false, helmEnv, args...)
		errorMsg := errBuffer.String()

		if len(errorMsg) > 0 {
			log.Entry(ctx).Infof(errorMsg)
		}

		if err != nil {
			return nil, helm.UserErr("std out err", fmt.Errorf(outBuffer.String(), fmt.Errorf(errorMsg)))
		}

		renderedManifests.Append(outBuffer.Bytes())
	}

	manifests, err := renderedManifests.SetLabels(h.labels, manifest.NewResourceSelectorLabels(h.transformAllowlist, h.transformDenylist))
	if err != nil {
		return nil, err
	}

	return manifests, nil
}
