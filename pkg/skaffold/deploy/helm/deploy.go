/*
Copyright 2019 The Skaffold Authors

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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver"
	backoff "github.com/cenkalti/backoff/v4"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/component/kubernetes"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kloader "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	kstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

var (
	// versionRegex extracts version from "helm version --client", for instance: "2.14.0-rc.2"
	versionRegex = regexp.MustCompile(`v?(\d[\w.\-]+)`)

	// helm32Version represents the version cut-off for helm3 behavior
	helm32Version = semver.MustParse("3.2.0")

	// helm31Version represents the version cut-off for helm3.1 post-renderer behavior
	helm31Version = semver.MustParse("3.1.0")

	// error to throw when helm version can't be determined
	versionErrorString = "failed to determine binary version: %w"

	// osExecutable allows for replacing the skaffold binary for testing purposes
	osExecutable = os.Executable
)

// for testing
var writeBuildArtifactsFunc = writeBuildArtifacts

// Deployer deploys workflows using the helm CLI
type Deployer struct {
	*latestV2.HelmDeploy

	accessor      access.Accessor
	debugger      debug.Debugger
	imageLoader   loader.ImageLoader
	logger        log.Logger
	statusMonitor status.Monitor
	syncer        sync.Syncer
	hookRunner    hooks.Runner

	podSelector *kubernetes.ImageList
	// originalImages []graph.Artifact // the set of images defined in ArtifactOverrides
	localImages []graph.Artifact // the set of images marked as "local" by the Runner

	kubeContext string
	kubeConfig  string
	namespace   string
	configFile  string

	namespaces *[]string

	// packaging temporary directory, used for predictable test output
	pkgTmpDir string

	labels map[string]string

	forceDeploy   bool
	enableDebug   bool
	isMultiConfig bool
	// bV is the helm binary version
	bV semver.Version
}

type Config interface {
	kubectl.Config
	kstatus.Config
	kloader.Config
	portforward.Config
	IsMultiConfig() bool
	JSONParseConfig() latestV2.JSONParseConfig
}

// NewDeployer returns a configured Deployer.  Returns an error if current version of helm is less than 3.1.0.
func NewDeployer(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latestV2.HelmDeploy) (*Deployer, error) {
	hv, err := binVer(ctx)
	if err != nil {
		return nil, versionGetErr(err)
	}

	if hv.LT(helm31Version) {
		return nil, minVersionErr(helm31Version.String())
	}

	podSelector := kubernetes.NewImageList()
	kubectl := pkgkubectl.NewCLI(cfg, cfg.GetKubeNamespace())
	namespaces, err := deployutil.GetAllPodNamespaces(cfg.GetNamespace(), cfg.GetPipelines())
	if err != nil {
		olog.Entry(context.TODO()).Warn("unable to parse namespaces - deploy might not work correctly!")
	}
	logger := component.NewLogger(cfg, kubectl, podSelector, &namespaces)
	return &Deployer{
		HelmDeploy:    h,
		podSelector:   podSelector,
		namespaces:    &namespaces,
		accessor:      component.NewAccessor(cfg, cfg.GetKubeContext(), kubectl, podSelector, labeller, &namespaces),
		debugger:      component.NewDebugger(cfg.Mode(), podSelector, &namespaces, cfg.GetKubeContext()),
		imageLoader:   component.NewImageLoader(cfg, kubectl),
		logger:        logger,
		statusMonitor: component.NewMonitor(cfg, cfg.GetKubeContext(), labeller, &namespaces),
		syncer:        component.NewSyncer(kubectl, &namespaces, logger.GetFormatter()),
		hookRunner:    hooks.NewDeployRunner(kubectl, h.LifecycleHooks, &namespaces, logger.GetFormatter(), hooks.NewDeployEnvOpts(labeller.GetRunID(), kubectl.KubeContext, namespaces)),
		kubeContext:   cfg.GetKubeContext(),
		kubeConfig:    cfg.GetKubeConfig(),
		namespace:     cfg.GetKubeNamespace(),
		forceDeploy:   cfg.ForceDeploy(),
		configFile:    cfg.ConfigurationFile(),
		labels:        labeller.Labels(),
		bV:            hv,
		enableDebug:   cfg.Mode() == config.RunModes.Debug,
		isMultiConfig: cfg.IsMultiConfig(),
	}, nil
}

func (h *Deployer) trackNamespaces(namespaces []string) {
	*h.namespaces = deployutil.ConsolidateNamespaces(*h.namespaces, namespaces)
}

func (h *Deployer) GetAccessor() access.Accessor {
	return h.accessor
}

func (h *Deployer) GetDebugger() debug.Debugger {
	return h.debugger
}

func (h *Deployer) GetLogger() log.Logger {
	return h.logger
}

func (h *Deployer) GetStatusMonitor() status.Monitor {
	return h.statusMonitor
}

func (h *Deployer) GetSyncer() sync.Syncer {
	return h.syncer
}

func (h *Deployer) RegisterLocalImages(images []graph.Artifact) {
	h.localImages = images
}

func (h *Deployer) TrackBuildArtifacts(artifacts []graph.Artifact) {
	deployutil.AddTagsToPodSelector(artifacts, h.localImages, h.podSelector)
	h.logger.RegisterArtifacts(artifacts)
}

// Deploy deploys the build results to the Kubernetes cluster
func (h *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy", map[string]string{
		"DeployerType": "helm",
	})
	defer endTrace()

	// Check that the cluster is reachable.
	// This gives a better error message when the cluster can't be reached.
	if err := kubernetes.FailIfClusterIsNotReachable(h.kubeContext); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_LoadImages")
	if err := h.imageLoader.LoadImages(childCtx, out, h.localImages, nil, builds); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	olog.Entry(ctx).Infof("Deploying with helm v%s ...", h.bV)

	nsMap := map[string]struct{}{}
	manifests := manifest.ManifestList{}

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
		m, results, err := h.deployRelease(ctx, out, releaseName, r, builds, h.bV, chartVersion)
		if err != nil {
			return userErr(fmt.Sprintf("deploying %q", releaseName), err)
		}

		manifests.Append(m)

		// collect namespaces
		for _, r := range results {
			if trimmed := strings.TrimSpace(r.Namespace); trimmed != "" {
				nsMap[trimmed] = struct{}{}
			}
		}
	}

	// Let's make sure that every image tag is set with `--set`.
	// Otherwise, templates have no way to use the images that were built.
	// Skip warning for multi-config projects as there can be artifacts without any usage in the current deployer.
	if !h.isMultiConfig {
		warnAboutUnusedImages(builds, manifests)
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

// Dependencies returns a list of files that the deployer depends on.
func (h *Deployer) Dependencies() ([]string, error) {
	var deps []string

	for _, release := range h.Releases {
		r := release
		deps = append(deps, r.ValuesFiles...)

		if r.ChartPath == "" {
			// chart path is only a dependency if it exists on the local filesystem
			continue
		}

		chartDepsDirs := []string{
			"charts",
			"tmpcharts",
		}

		lockFiles := []string{
			"Chart.lock",
		}

		// We can always add a dependency if it is not contained in our chartDepsDirs.
		// However, if the file is in our chartDepsDir, we can only include the file
		// if we are not running the helm dep build phase, as that modifies files inside
		// the chartDepsDir and results in an infinite build loop.
		// We additionally exclude ChartFile.lock,
		// since it also gets modified during a `helm dep build`.
		isDep := func(path string, info walk.Dirent) (bool, error) {
			if info.IsDir() {
				return false, nil
			}
			if r.SkipBuildDependencies {
				return true, nil
			}

			for _, v := range chartDepsDirs {
				if strings.HasPrefix(path, filepath.Join(release.ChartPath, v)) {
					return false, nil
				}
			}

			for _, v := range lockFiles {
				if strings.EqualFold(info.Name(), v) {
					return false, nil
				}
			}

			return true, nil
		}

		if err := walk.From(release.ChartPath).When(isDep).AppendPaths(&deps); err != nil {
			return deps, userErr("issue walking releases", err)
		}
	}
	sort.Strings(deps)
	return deps, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (h *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "helm",
	})

	var errMsgs []string
	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return fmt.Errorf("cannot parse the release name template: %w", err)
		}

		namespace, err := h.releaseNamespace(r)
		if err != nil {
			return err
		}
		args := []string{}
		if dryRun {
			args = append(args, "get", "manifest")
		} else {
			args = append(args, "delete")
		}
		args = append(args, releaseName)

		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}
		if err := h.exec(ctx, out, false, nil, args...); err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
	}

	if len(errMsgs) != 0 {
		return deployerr.CleanupErr(fmt.Errorf(strings.Join(errMsgs, "\n")))
	}
	return nil
}

// Render generates the Kubernetes manifests and writes them out
func (h *Deployer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "helm",
	})
	renderedManifests := new(bytes.Buffer)
	helmEnv := util.OSEnviron()
	var postRendererArgs []string

	if len(builds) > 0 {
		skaffoldBinary, filterEnv, cleanup, err := h.prepareSkaffoldFilter(builds)
		if err != nil {
			return fmt.Errorf("could not prepare `skaffold filter`: %w", err)
		}
		// need to include current environment, specifically for HOME to lookup ~/.kube/config
		helmEnv = append(helmEnv, filterEnv...)
		postRendererArgs = []string{"--post-renderer", skaffoldBinary}
		defer cleanup()
	}

	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return userErr(fmt.Sprintf("cannot expand release name %q", r.Name), err)
		}

		args := []string{"template", releaseName, chartSource(r)}
		args = append(args, postRendererArgs...)
		if r.Packaged == nil && r.Version != "" {
			args = append(args, "--version", r.Version)
		}

		args, err = constructOverrideArgs(&r, builds, args)
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
		if err := h.exec(ctx, outBuffer, false, helmEnv, args...); err != nil {
			return userErr("std out err", fmt.Errorf(outBuffer.String()))
		}
		renderedManifests.Write(outBuffer.Bytes())
	}

	return manifest.Write(renderedManifests.String(), filepath, out)
}

func (h *Deployer) HasRunnableHooks() bool {
	return len(h.HelmDeploy.LifecycleHooks.PreHooks) > 0 || len(h.HelmDeploy.LifecycleHooks.PostHooks) > 0
}

func (h *Deployer) PreDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PreHooks")
	if err := h.hookRunner.RunPreHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (h *Deployer) PostDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PostHooks")
	if err := h.hookRunner.RunPostHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

// deployRelease deploys a single release; returns the deployed manifests, and the artifacts
func (h *Deployer) deployRelease(ctx context.Context, out io.Writer, releaseName string, r latestV2.HelmRelease, builds []graph.Artifact, helmVersion semver.Version, chartVersion string) ([]byte, []types.Artifact, error) {
	var err error
	opts := installOpts{
		releaseName: releaseName,
		upgrade:     true,
		flags:       h.Flags.Upgrade,
		force:       h.forceDeploy,
		chartPath:   chartSource(r),
		helmVersion: helmVersion,
		repo:        r.Repo,
		version:     chartVersion,
	}

	installEnv := util.OSEnviron()
	if len(builds) > 0 {
		skaffoldBinary, filterEnv, cleanup, err := h.prepareSkaffoldFilter(builds)
		if err != nil {
			return nil, nil, fmt.Errorf("could not prepare `skaffold filter`: %w", err)
		}

		// need to include current environment, specifically for HOME to lookup ~/.kube/config
		installEnv = append(installEnv, filterEnv...)
		opts.postRenderer = skaffoldBinary
		defer cleanup()
	}
	opts.namespace, err = h.releaseNamespace(r)
	if err != nil {
		return nil, nil, err
	}

	if err := h.exec(ctx, ioutil.Discard, false, nil, getArgs(releaseName, opts.namespace)...); err != nil {
		output.Yellow.Fprintf(out, "Helm release %s not installed. Installing...\n", releaseName)

		opts.upgrade = false
		opts.flags = h.Flags.Install
	} else {
		if r.UpgradeOnChange != nil && !*r.UpgradeOnChange {
			olog.Entry(ctx).Infof("Release %s already installed...", releaseName)
			return nil, []types.Artifact{}, nil
		} else if r.UpgradeOnChange == nil && r.RemoteChart != "" {
			olog.Entry(ctx).Infof("Release %s not upgraded as it is remote...", releaseName)
			return nil, []types.Artifact{}, nil
		}
	}

	// Only build local dependencies, but allow a user to skip them.
	if !r.SkipBuildDependencies && r.ChartPath != "" {
		olog.Entry(ctx).Info("Building helm dependencies...")

		if err := h.exec(ctx, out, false, nil, "dep", "build", r.ChartPath); err != nil {
			return nil, nil, userErr("building helm dependencies", err)
		}
	}

	// Dump overrides to a YAML file to pass into helm
	if len(r.Overrides.Values) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return nil, nil, userErr("cannot marshal overrides to create overrides values.yaml", err)
		}

		if err := ioutil.WriteFile(constants.HelmOverridesFilename, overrides, 0666); err != nil {
			return nil, nil, userErr(fmt.Sprintf("cannot create file %q", constants.HelmOverridesFilename), err)
		}

		defer func() {
			os.Remove(constants.HelmOverridesFilename)
		}()
	}

	if r.Packaged != nil {
		chartPath, err := h.packageChart(ctx, r)
		if err != nil {
			return nil, nil, userErr("cannot package chart", err)
		}

		opts.chartPath = chartPath
	}

	args, err := h.installArgs(r, builds, opts)
	if err != nil {
		return nil, nil, userErr("release args", err)
	}

	err = h.exec(ctx, out, r.UseHelmSecrets, installEnv, args...)
	if err != nil {
		return nil, nil, userErr("install", err)
	}

	// get the kubernetes manifests deployed to the cluster
	b, err := h.getReleaseManifest(ctx, releaseName, opts.namespace)
	if err != nil {
		return nil, nil, userErr("get release", err)
	}
	artifacts := parseReleaseManifests(opts.namespace, bufio.NewReader(bytes.NewReader(b)))
	return b, artifacts, nil
}

// getReleaseManifest confirms that a release is visible to helm and returns the release manifest
func (h *Deployer) getReleaseManifest(ctx context.Context, releaseName string, namespace string) ([]byte, error) {
	// Retry, because sometimes a release may not be immediately visible
	opts := backoff.NewExponentialBackOff()
	opts.MaxElapsedTime = 4 * time.Second
	var b bytes.Buffer

	err := backoff.Retry(
		func() error {
			// only intereted in the deployed YAML
			args := getArgs(releaseName, namespace)
			args = append(args, "--template", "{{.Release.Manifest}}")
			if err := h.exec(ctx, &b, false, nil, args...); err != nil {
				olog.Entry(ctx).Debugf("unable to get release: %v (may retry):\n%s", err, b.String())
				return err
			}
			return nil
		}, opts)

	olog.Entry(ctx).Debug(b.String())

	return b.Bytes(), err
}

// packageChart packages the chart and returns the path to the resulting chart archive
func (h *Deployer) packageChart(ctx context.Context, r latestV2.HelmRelease) (string, error) {
	// Allow a test to sneak a predictable path in
	tmpDir := h.pkgTmpDir

	if tmpDir == "" {
		t, err := ioutil.TempDir("", "skaffold-helm")
		if err != nil {
			return "", fmt.Errorf("tempdir: %w", err)
		}
		tmpDir = t
	}

	args := []string{"package", r.ChartPath, "--destination", tmpDir}

	if r.Packaged.Version != "" {
		v, err := util.ExpandEnvTemplate(r.Packaged.Version, nil)
		if err != nil {
			return "", fmt.Errorf("packaged.version template: %w", err)
		}
		args = append(args, "--version", v)
	}

	if r.Packaged.AppVersion != "" {
		av, err := util.ExpandEnvTemplate(r.Packaged.AppVersion, nil)
		if err != nil {
			return "", fmt.Errorf("packaged.appVersion template: %w", err)
		}
		args = append(args, "--app-version", av)
	}

	buf := &bytes.Buffer{}

	if err := h.exec(ctx, buf, false, nil, args...); err != nil {
		return "", fmt.Errorf("package chart into a .tgz archive: %v: %w", args, err)
	}

	output := strings.TrimSpace(buf.String())
	idx := strings.Index(output, tmpDir)

	if idx == -1 {
		return "", fmt.Errorf("unable to find %s in output: %s", tmpDir, output)
	}

	return output[idx:], nil
}

func chartSource(r latestV2.HelmRelease) string {
	if r.RemoteChart != "" {
		return r.RemoteChart
	}
	return r.ChartPath
}

func warnAboutUnusedImages(builds []graph.Artifact, manifests manifest.ManifestList) {
	seen := map[string]bool{}
	images, _ := manifests.GetImages()
	for _, a := range images {
		seen[a.Tag] = true
	}
	for _, b := range builds {
		if !seen[b.Tag] {
			warnings.Printf("image [%s] is not used.", b.Tag)
			warnings.Printf("See helm documentation on how to replace image names with their actual tags: https://skaffold.dev/docs/pipeline-stages/deployers/helm/#image-configuration")
		}
	}
}
