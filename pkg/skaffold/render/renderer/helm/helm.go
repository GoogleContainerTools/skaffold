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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/generate"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	sUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type Config interface {
	render.Config
	Mode() config.RunMode
	ConfigurationFile() string
	GetKubeContext() string
	GetKubeConfig() string
}

type Helm struct {
	generate.Generator
	config *latest.Helm

	kubeContext string
	kubeConfig  string
	namespace   string
	configFile  string
	labels      map[string]string
	enableDebug bool

	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (h Helm) EnableDebug() bool         { return h.enableDebug }
func (h Helm) ConfigFile() string        { return h.configFile }
func (h Helm) KubeContext() string       { return h.kubeContext }
func (h Helm) KubeConfig() string        { return h.kubeConfig }
func (h Helm) Labels() map[string]string { return h.labels }
func (h Helm) GlobalFlags() []string     { return h.config.Flags.Global }

func New(cfg Config, rCfg latest.RenderConfig, labels map[string]string) (Helm, error) {
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate)
	transformAllowlist, transformDenylist, err := util.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return Helm{}, err
	}
	return Helm{
		Generator: generator,
		config:    rCfg.Helm,

		enableDebug: cfg.Mode() == config.RunModes.Debug,
		configFile:  cfg.ConfigurationFile(),
		kubeContext: cfg.GetKubeContext(),
		kubeConfig:  cfg.GetKubeConfig(),
		labels:      labels,

		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (h Helm) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool) (manifest.ManifestList, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "Render_HelmManifests")
	log.Entry(ctx).Infof("rendering using helm")
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"RendererType": "helm",
	})

	manifests, err := h.generateHelmManifests(ctx, builds)
	endTrace()
	return manifests, err
}

func (h Helm) generateHelmManifests(ctx context.Context, builds []graph.Artifact) (manifest.ManifestList, error) {
	renderedManifests := new(bytes.Buffer)
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

		args := []string{"template", releaseName, helm.ChartSource(release)}
		args = append(args, postRendererArgs...)
		if release.Packaged == nil && release.Version != "" {
			args = append(args, "--version", release.Version)
		}

		args, err = helm.ConstructOverrideArgs(&release, builds, args)
		if err != nil {
			return nil, helm.UserErr("construct override args", err)
		}

		namespace, err := helm.ReleaseNamespace(h.namespace, release)
		if err != nil {
			return nil, err
		}
		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}

		if release.Repo != "" {
			args = append(args, "--repo")
			args = append(args, release.Repo)
		}

		outBuffer := new(bytes.Buffer)
		if err := helm.Exec(ctx, h, outBuffer, false, helmEnv, args...); err != nil {
			return nil, helm.UserErr("std out err", fmt.Errorf(outBuffer.String()))
		}
		renderedManifests.Write(outBuffer.Bytes())
	}
	manifests, err := manifest.Load(bytes.NewReader(renderedManifests.Bytes()))
	if err != nil {
		return nil, err
	}

	manifests, err = manifests.SetLabels(h.labels, manifest.NewResourceSelectorLabels(h.transformAllowlist, h.transformDenylist))
	if err != nil {
		return nil, err
	}

	return manifests, nil
}
