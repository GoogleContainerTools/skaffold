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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/blang/semver"
	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

// Deployer30 deploys workflows using the helm CLI less than 3.1
type Deployer30 struct {
	*Deployer3
}

func NewDeployer30(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latest.HelmDeploy, hv semver.Version) (*Deployer30, error) {
	d3, err := NewBase(ctx, cfg, labeller, h, hv)
	if err != nil {
		return nil, err
	}
	return &Deployer30{
		Deployer3: d3,
	}, nil
}

// Deploy deploys the build results to the Kubernetes cluster
func (h *Deployer30) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy", map[string]string{
		"DeployerType": "helm30",
	})
	defer endTrace()

	// Check that the cluster is reachable.
	// This gives a better error message when the cluster can't
	// be reached.
	if err := kubernetes.FailIfClusterIsNotReachable(h.kubeContext); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_LoadImages")
	if err := h.imageLoader.LoadImages(childCtx, out, h.localImages, h.originalImages, builds); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	olog.Entry(ctx).Infof("Deploying with helm v%s ...", h.bV)

	var dRes []types.Artifact
	nsMap := map[string]struct{}{}
	valuesSet := map[string]bool{}

	// Deploy every release
	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return userErr(fmt.Sprintf("cannot expand release name %q", r.Name), err)
		}
		chartVersion, err := util.ExpandEnvTemplateOrFail(r.Version, nil)
		if err != nil {
			return userErr(fmt.Sprintf("cannot expand chart version %q", r.Version), err)
		}
		results, err := h.deployRelease(ctx, out, releaseName, r, builds, valuesSet, h.bV, chartVersion)
		if err != nil {
			return userErr(fmt.Sprintf("deploying %q", releaseName), err)
		}

		// collect namespaces
		for _, r := range results {
			if trimmed := strings.TrimSpace(r.Namespace); trimmed != "" {
				nsMap[trimmed] = struct{}{}
			}
		}

		dRes = append(dRes, results...)
	}

	// Let's make sure that every image tag is set with `--set`.
	// Otherwise, templates have no way to use the images that were built.
	// Skip warning for multi-config projects as there can be artifacts without any usage in the current deployer.
	if !h.isMultiConfig {
		warnAboutUnusedImages(builds, valuesSet)
	}

	if err := label.Apply(ctx, h.labels, dRes, h.kubeContext); err != nil {
		return helmLabelErr(fmt.Errorf("adding labels: %w", err))
	}

	// Collect namespaces in a string
	var namespaces []string
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}

	h.TrackBuildArtifacts(builds)
	h.trackNamespaces(namespaces)
	return nil
}

// deployRelease deploys a single release
func (h *Deployer30) deployRelease(ctx context.Context, out io.Writer, releaseName string, r latest.HelmRelease, builds []graph.Artifact, valuesSet map[string]bool, helmVersion semver.Version, chartVersion string) ([]types.Artifact, error) {
	var err error
	opts := h.getInstallOpts(releaseName, r, helmVersion, chartVersion)

	var installEnv []string
	if h.enableDebug {
		if h.bV.LT(helm31Version) {
			return nil, fmt.Errorf("debug requires at least Helm 3.1 (current: %v)", h.bV)
		}
		var binary string
		if binary, err = osExecutable(); err != nil {
			return nil, fmt.Errorf("cannot locate this Skaffold binary: %w", err)
		}
		opts.postRenderer = binary

		var buildsFile string
		if len(builds) > 0 {
			var cleanup func()
			buildsFile, cleanup, err = writeBuildArtifacts(builds)
			if err != nil {
				return nil, fmt.Errorf("could not write build-artifacts: %w", err)
			}
			defer cleanup()
		}

		cmdLine := h.generateSkaffoldDebugFilter(buildsFile)

		// need to include current environment, specifically for HOME to lookup ~/.kube/config
		env := util.EnvSliceToMap(util.OSEnviron(), "=")
		env["SKAFFOLD_CMDLINE"] = shell.Join(cmdLine...)
		env["SKAFFOLD_FILENAME"] = h.configFile
		installEnv = util.EnvMapToSlice(env, "=")
	}

	opts.namespace, err = h.releaseNamespace(r)
	if err != nil {
		return nil, err
	}

	if err := h.exec(ctx, ioutil.Discard, false, nil, getArgs(releaseName, opts.namespace)...); err != nil {
		output.Yellow.Fprintf(out, "Helm release %s not installed. Installing...\n", releaseName)

		opts.upgrade = false
		opts.flags = h.Flags.Install
	} else {
		if r.UpgradeOnChange != nil && !*r.UpgradeOnChange {
			olog.Entry(ctx).Infof("Release %s already installed...", releaseName)
			return []types.Artifact{}, nil
		} else if r.UpgradeOnChange == nil && r.RemoteChart != "" {
			olog.Entry(ctx).Infof("Release %s not upgraded as it is remote...", releaseName)
			return []types.Artifact{}, nil
		}
	}

	// Only build local dependencies, but allow a user to skip them.
	if !r.SkipBuildDependencies && r.ChartPath != "" {
		olog.Entry(ctx).Info("Building helm dependencies...")

		if err := h.exec(ctx, out, false, nil, "dep", "build", r.ChartPath); err != nil {
			return nil, userErr("building helm dependencies", err)
		}
	}

	// Dump overrides to a YAML file to pass into helm
	if len(r.Overrides.Values) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return nil, userErr("cannot marshal overrides to create overrides values.yaml", err)
		}

		if err := ioutil.WriteFile(constants.HelmOverridesFilename, overrides, 0666); err != nil {
			return nil, userErr(fmt.Sprintf("cannot create file %q", constants.HelmOverridesFilename), err)
		}

		defer func() {
			os.Remove(constants.HelmOverridesFilename)
		}()
	}

	if r.Packaged != nil {
		chartPath, err := h.packageChart(ctx, r)
		if err != nil {
			return nil, userErr("cannot package chart", err)
		}

		opts.chartPath = chartPath
	}

	args, err := h.installArgs(r, builds, valuesSet, opts)
	if err != nil {
		return nil, userErr("release args", err)
	}

	err = h.exec(ctx, out, r.UseHelmSecrets, installEnv, args...)
	if err != nil {
		return nil, userErr("install", err)
	}

	b, err := h.getReleaseManifest(ctx, releaseName, opts.namespace)
	if err != nil {
		return nil, userErr("get release", err)
	}

	artifacts := parseReleaseManifests(opts.namespace, bufio.NewReader(&b))
	return artifacts, nil
}

// Render generates the Kubernetes manifests and writes them out
func (h *Deployer30) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "helm",
	})
	renderedManifests := new(bytes.Buffer)

	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return userErr(fmt.Sprintf("cannot expand release name %q", r.Name), err)
		}

		args := []string{"template", releaseName, chartSource(r)}
		if r.Packaged == nil && r.Version != "" {
			args = append(args, "--version", r.Version)
		}

		params, err := pairParamsToArtifacts(builds, r.ArtifactOverrides)
		if err != nil {
			return err
		}

		for k, v := range params {
			var value string

			cfg := r.ImageStrategy.HelmImageConfig.HelmConventionConfig

			value, err = imageSetFromConfig(cfg, k, v.Tag)
			if err != nil {
				return err
			}

			args = append(args, "--set-string", value)
		}

		args, err = constructOverrideArgs(&r, builds, args, func(string) {})
		if err != nil {
			return userErr("construct override args", err)
		}

		namespace, err := h.releaseNamespace(r)
		if err != nil {
			return err
		}
		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}

		if r.Repo != "" {
			args = append(args, "--repo")
			args = append(args, r.Repo)
		}

		outBuffer := new(bytes.Buffer)
		if err := h.exec(ctx, outBuffer, false, nil, args...); err != nil {
			return userErr("std out err", fmt.Errorf(outBuffer.String()))
		}
		renderedManifests.Write(outBuffer.Bytes())
	}

	return manifest.Write(renderedManifests.String(), filepath, out)
}

func (h *Deployer30) generateSkaffoldDebugFilter(buildsFile string) []string {
	args := []string{"filter", "--debugging", "--kube-context", h.kubeContext}
	if len(buildsFile) > 0 {
		args = append(args, "--build-artifacts", buildsFile)
	}
	args = append(args, h.Flags.Global...)

	if h.kubeConfig != "" {
		args = append(args, "--kubeconfig", h.kubeConfig)
	}
	return args
}
