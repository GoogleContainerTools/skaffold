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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	sync2 "sync"
	"time"

	"github.com/blang/semver"
	"github.com/cenkalti/backoff/v4"
	"golang.org/x/sync/errgroup"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/component/kubernetes"
	deployerr "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/types"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	kloader "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/loader"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/portforward"
	kstatus "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	renderutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

var (
	// helm32Version represents the version cut-off for helm3 behavior
	helm32Version = semver.MustParse("3.2.0")

	// helm31Version represents the version cut-off for helm3.1 post-renderer behavior
	helm31Version = semver.MustParse("3.1.0")
)

// Deployer deploys workflows using the helm CLI
type Deployer struct {
	configName string

	*latest.LegacyHelmDeploy

	accessor      access.Accessor
	debugger      debug.Debugger
	imageLoader   loader.ImageLoader
	logger        log.Logger
	statusMonitor status.Monitor
	syncer        sync.Syncer
	hookRunner    hooks.Runner

	podSelector    *kubernetes.ImageList
	originalImages []graph.Artifact // the set of images defined in ArtifactOverrides
	localImages    []graph.Artifact // the set of images marked as "local" by the Runner

	kubeContext string
	kubeConfig  string
	namespace   string
	configFile  string

	namespaces          *[]string
	manifestsNamespaces *[]string

	// packaging temporary directory, used for predictable test output
	pkgTmpDir string

	labels map[string]string

	forceDeploy       bool
	enableDebug       bool
	overrideProtocols []string
	isMultiConfig     bool
	helmVersion       semver.Version

	transformableAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformableDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (h Deployer) ManifestOverrides() map[string]string {
	return map[string]string{}
}

func (h Deployer) EnableDebug() bool           { return h.enableDebug }
func (h Deployer) OverrideProtocols() []string { return h.overrideProtocols }
func (h Deployer) ConfigFile() string          { return h.configFile }
func (h Deployer) KubeContext() string         { return h.kubeContext }
func (h Deployer) KubeConfig() string          { return h.kubeConfig }
func (h Deployer) Labels() map[string]string   { return h.labels }
func (h Deployer) GlobalFlags() []string       { return h.LegacyHelmDeploy.Flags.Global }

type Config interface {
	kubectl.Config
	kstatus.Config
	kloader.Config
	portforward.Config
	GetNamespace() string
	IsMultiConfig() bool
	JSONParseConfig() latest.JSONParseConfig
}

// NewDeployer returns a configured Deployer.  Returns an error if current version of helm is less than 3.1.0.
func NewDeployer(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latest.LegacyHelmDeploy, artifacts []*latest.Artifact, configName string, customResourceSelectors []manifest.GroupKindSelector) (*Deployer, error) {
	helmVersion, err := helm.BinVer(ctx)
	if err != nil {
		return nil, helm.VersionGetErr(err)
	}

	if helmVersion.LT(helm31Version) {
		return nil, helm.MinVersionErr(helm31Version.String())
	}

	podSelector := kubernetes.NewImageList()
	kubectl := pkgkubectl.NewCLI(cfg, cfg.GetKubeNamespace())
	namespaces, err := deployutil.GetAllPodNamespaces(cfg.GetNamespace(), cfg.GetPipelines())
	if err != nil {
		olog.Entry(context.TODO()).Warn("unable to parse namespaces - deploy might not work correctly!")
	}
	logger := component.NewLogger(cfg, kubectl, podSelector, &namespaces)
	transformableAllowlist, transformableDenylist, err := renderutil.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return nil, err
	}
	var ogImages []graph.Artifact
	for _, artifact := range artifacts {
		ogImages = append(ogImages, graph.Artifact{
			ImageName:   artifact.ImageName,
			RuntimeType: artifact.RuntimeType,
		})
	}

	manifestsNamespaces := []string{}

	return &Deployer{
		configName:             configName,
		LegacyHelmDeploy:       h,
		podSelector:            podSelector,
		namespaces:             &namespaces,
		accessor:               component.NewAccessor(cfg, cfg.GetKubeContext(), kubectl, podSelector, labeller, &namespaces),
		debugger:               component.NewDebugger(cfg.Mode(), podSelector, &namespaces, cfg.GetKubeContext()),
		imageLoader:            component.NewImageLoader(cfg, kubectl),
		logger:                 logger,
		statusMonitor:          component.NewMonitor(cfg, cfg.GetKubeContext(), labeller, &namespaces, customResourceSelectors),
		syncer:                 component.NewSyncer(kubectl, &namespaces, logger.GetFormatter()),
		manifestsNamespaces:    &manifestsNamespaces,
		hookRunner:             hooks.NewDeployRunner(kubectl, h.LifecycleHooks, &namespaces, logger.GetFormatter(), hooks.NewDeployEnvOpts(labeller.GetRunID(), kubectl.KubeContext, namespaces), &manifestsNamespaces),
		originalImages:         ogImages,
		kubeContext:            cfg.GetKubeContext(),
		kubeConfig:             cfg.GetKubeConfig(),
		namespace:              cfg.GetKubeNamespace(),
		forceDeploy:            cfg.ForceDeploy(),
		configFile:             cfg.ConfigurationFile(),
		labels:                 labeller.Labels(),
		helmVersion:            helmVersion,
		enableDebug:            cfg.Mode() == config.RunModes.Debug,
		overrideProtocols:      debug.Protocols,
		isMultiConfig:          cfg.IsMultiConfig(),
		transformableAllowlist: transformableAllowlist,
		transformableDenylist:  transformableDenylist,
	}, nil
}

func (h *Deployer) ConfigName() string {
	return h.configName
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

func (h *Deployer) TrackBuildArtifacts(builds, deployedImages []graph.Artifact) {
	deployutil.AddTagsToPodSelector(builds, deployedImages, h.podSelector)
	h.logger.RegisterArtifacts(builds)
}

// Deploy deploys the build results to the Kubernetes cluster
func (h *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact, _ manifest.ManifestListByConfig) error {
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
	if err := h.imageLoader.LoadImages(childCtx, out, h.localImages, h.originalImages, builds); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	olog.Entry(ctx).Infof("Deploying with helm v%s ...", h.helmVersion)

	dependencyGraph, err := NewDependencyGraph(h.Releases)
	if err != nil {
		return fmt.Errorf("unable to create dependency graph: %w", err)
	}

	levelByLevelReleases, err := dependencyGraph.GetReleasesByLevel()
	if err != nil {
		return fmt.Errorf("unable to get releases by level: %w", err)
	}

	var mu sync2.Mutex
	nsMap := map[string]struct{}{}
	manifests := manifest.ManifestList{}

	concurrency := 1
	if h.Concurrency != nil {
		if *h.Concurrency == 0 {
			concurrency = -1 // unlimited
		} else {
			concurrency = *h.Concurrency
		}
	}

	if concurrency == 1 {
		olog.Entry(ctx).Infof("Installing %d releases sequentially", len(h.Releases))
	} else {
		olog.Entry(ctx).Infof("Installing %d releases concurrently", len(h.Releases))
	}

	releaseNameToRelease := make(map[string]latest.HelmRelease)
	for _, r := range h.Releases {
		releaseNameToRelease[r.Name] = r
	}

	levels := make([]int, 0, len(levelByLevelReleases))
	for level := range levelByLevelReleases {
		levels = append(levels, level)
	}
	// Sort levels in ascending order
	sort.Ints(levels)

	// Process each level in order
	for _, level := range levels {
		releases := levelByLevelReleases[level]
		if len(levelByLevelReleases) > 1 {
			olog.Entry(ctx).Infof("Installing level %d/%d releases (%d releases)", level+1, len(levelByLevelReleases), len(releases))
		} else {
			olog.Entry(ctx).Infof("Installing releases (%d releases)", len(releases))
		}

		g, levelCtx := errgroup.WithContext(ctx)
		g.SetLimit(concurrency)
		// Deploy releases in current level
		for _, name := range releases {
			release := releaseNameToRelease[name]

			g.Go(func() error {
				chartVersion, err := util.ExpandEnvTemplateOrFail(release.Version, nil)
				if err != nil {
					return helm.UserErr(fmt.Sprintf("cannot expand chart version %q", release.Version), err)
				}

				repo, err := util.ExpandEnvTemplateOrFail(release.Repo, nil)
				if err != nil {
					return helm.UserErr(fmt.Sprintf("cannot expand repo %q", release.Repo), err)
				}

				release.ChartPath, err = util.ExpandEnvTemplateOrFail(release.ChartPath, nil)
				if err != nil {
					return helm.UserErr(fmt.Sprintf("cannot expand chart path %q", release.ChartPath), err)
				}

				releaseName, err := util.ExpandEnvTemplateOrFail(release.Name, nil)
				if err != nil {
					return helm.UserErr(fmt.Sprintf("cannot expand release name %q", release.Name), err)
				}

				m, results, err := h.deployRelease(levelCtx, out, releaseName, release, builds, h.helmVersion, chartVersion, repo)
				if err != nil {
					return helm.UserErr(fmt.Sprintf("deploying %q", releaseName), err)
				}

				mu.Lock()
				defer mu.Unlock()
				manifests.Append(m)
				for _, res := range results {
					if trimmed := strings.TrimSpace(res.Namespace); trimmed != "" {
						nsMap[trimmed] = struct{}{}
					}
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	// Let's make sure that every image tag is set with `--set`.
	// Otherwise, templates have no way to use the images that were built.
	// Skip warning for multi-config projects as there can be artifacts without any usage in the current deployer.
	if !h.isMultiConfig {
		h.warnAboutUnusedImages(builds, manifests)
	}

	// Collect namespaces in a string
	var namespaces []string
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}
	deployedImages, _ := manifests.GetImages(manifest.NewResourceSelectorImages(h.transformableAllowlist, h.transformableDenylist))

	h.TrackBuildArtifacts(builds, deployedImages)
	h.trackNamespaces(namespaces)
	*h.manifestsNamespaces = namespaces
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

		expandedPath, e := util.ExpandEnvTemplateOrFail(release.ChartPath, nil)
		if e != nil {
			return deps, helm.UserErr("issue expanding variable", e)
		}
		if err := walk.From(expandedPath).When(isDep).AppendPaths(&deps); err != nil {
			return deps, helm.UserErr("issue walking releases", err)
		}
	}
	sort.Strings(deps)
	return deps, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (h *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool, _ manifest.ManifestListByConfig) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "helm",
	})

	var errMsgs []string
	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return fmt.Errorf("cannot parse the release name template: %w", err)
		}

		namespace, err := helm.ReleaseNamespace(h.namespace, r)
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
		if err := helm.Exec(ctx, h, out, false, nil, args...); err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
	}

	if len(errMsgs) != 0 {
		return deployerr.CleanupErr(errors.New(strings.Join(errMsgs, "\n")))
	}
	return nil
}

func (h *Deployer) HasRunnableHooks() bool {
	return len(h.LegacyHelmDeploy.LifecycleHooks.PreHooks) > 0 || len(h.LegacyHelmDeploy.LifecycleHooks.PostHooks) > 0
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
func (h *Deployer) deployRelease(ctx context.Context, out io.Writer, releaseName string, r latest.HelmRelease, builds []graph.Artifact, helmVersion semver.Version, chartVersion string, repo string) ([]byte, []types.Artifact, error) {
	var err error
	opts := installOpts{
		releaseName: releaseName,
		upgrade:     true,
		flags:       h.Flags.Upgrade,
		force:       h.forceDeploy,
		chartPath:   helm.ChartSource(r),
		helmVersion: helmVersion,
		repo:        repo,
		version:     chartVersion,
	}

	opts.namespace, err = helm.ReleaseNamespace(h.namespace, r)
	if err != nil {
		return nil, nil, err
	}

	if err := helm.Exec(ctx, h, io.Discard, false, nil, helm.GetArgs(releaseName, opts.namespace)...); err != nil {
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

	installEnv := util.OSEnviron()
	// skaffold use the post-renderer feature to do skaffold specific rendering such as image replacement, adding debugging annotation in helm rendered result,
	// as Helm doesn't support to run multiple post-renderers,  this is used to run user-defined render inside skaffold filter which happens before skaffold
	// post-rendering process for helm releases.
	postRendererFlag := getPostRendererFlag(opts.flags)
	skaffoldBinary, filterEnv, cleanup, err := helm.PrepareSkaffoldFilter(h, builds, postRendererFlag)
	if err != nil {
		return nil, nil, fmt.Errorf("could not prepare `skaffold filter`: %w", err)
	}

	if cleanup != nil {
		defer cleanup()
	}
	// need to include current environment, specifically for HOME to lookup ~/.kube/config
	installEnv = append(installEnv, filterEnv...)

	// Only build local dependencies, but allow a user to skip them.
	if !r.SkipBuildDependencies && r.ChartPath != "" {
		olog.Entry(ctx).Info("Building helm dependencies...")

		args := []string{"dep", "build", r.ChartPath}
		args = append(args, h.Flags.DepBuild...)

		if err := helm.Exec(ctx, h, out, false, nil, args...); err != nil {
			return nil, nil, helm.UserErr("building helm dependencies", err)
		}
	}

	// Dump overrides to a YAML file to pass into helm
	if len(r.Overrides.Values) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return nil, nil, helm.UserErr("cannot marshal overrides to create overrides values.yaml", err)
		}

		if err := os.WriteFile(constants.HelmOverridesFilename, overrides, 0666); err != nil {
			return nil, nil, helm.UserErr(fmt.Sprintf("cannot create file %q", constants.HelmOverridesFilename), err)
		}

		defer func() {
			if err := os.Remove(constants.HelmOverridesFilename); err != nil {
				olog.Entry(ctx).Debugf("unable to remove %q: %v", constants.HelmOverridesFilename, err)
			}
		}()
	}

	if r.Packaged != nil {
		chartPath, err := h.packageChart(ctx, r)
		if err != nil {
			return nil, nil, helm.UserErr("cannot package chart", err)
		}

		opts.chartPath = chartPath
	}

	args, err := h.installArgs(r, builds, opts)
	if err != nil {
		return nil, nil, helm.UserErr("release args", err)
	}

	cleanUpPostRenderer, postRendererArgs, err := helm.PreparePostRenderer(ctx, h, skaffoldBinary, h.helmVersion)
	if err != nil {
		return nil, nil, err
	}
	args = append(args, postRendererArgs...)
	if cleanUpPostRenderer != nil {
		defer cleanUpPostRenderer()
	}

	err = helm.Exec(ctx, h, out, r.UseHelmSecrets, installEnv, args...)
	if err != nil {
		return nil, nil, helm.UserErr("install", err)
	}

	// get the kubernetes manifests deployed to the cluster
	b, err := h.getReleaseManifest(ctx, releaseName, opts.namespace)
	if err != nil {
		return nil, nil, helm.UserErr("get release", err)
	}
	artifacts := parseReleaseManifests(opts.namespace, bufio.NewReader(bytes.NewReader(b)))
	return b, artifacts, nil
}

func getPostRendererFlag(flags []string) []string {
	for i, ele := range flags {
		if strings.HasPrefix(ele, "--post-renderer") {
			// "--post-renderer", "executable"
			if ele == "--post-renderer" {
				return []string{ele, flags[i+1]}
			}
			// "--post-renderer=executable"
			return []string{ele}
		}
	}
	return []string{}
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
			args := helm.GetArgs(releaseName, namespace)
			args = append(args, "--template", "{{.Release.Manifest}}")
			if err := helm.Exec(ctx, h, &b, false, nil, args...); err != nil {
				olog.Entry(ctx).Debugf("unable to get release: %v (may retry):\n%s", err, b.String())
				return err
			}
			return nil
		}, opts)

	olog.Entry(ctx).Debug(b.String())

	return b.Bytes(), err
}

// packageChart packages the chart and returns the path to the resulting chart archive
func (h *Deployer) packageChart(ctx context.Context, r latest.HelmRelease) (string, error) {
	// Allow a test to sneak a predictable path in
	tmpDir := h.pkgTmpDir

	if tmpDir == "" {
		t, err := os.MkdirTemp("", "skaffold-helm")
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

	if err := helm.Exec(ctx, h, buf, false, nil, args...); err != nil {
		return "", fmt.Errorf("package chart into a .tgz archive: %v: %w", args, err)
	}

	output := strings.TrimSpace(buf.String())
	idx := strings.Index(output, tmpDir)

	if idx == -1 {
		return "", fmt.Errorf("unable to find %s in output: %s", tmpDir, output)
	}

	return output[idx:], nil
}

func (h *Deployer) warnAboutUnusedImages(builds []graph.Artifact, manifests manifest.ManifestList) {
	seen := map[string]bool{}
	images, _ := manifests.GetImages(manifest.NewResourceSelectorImages(h.transformableAllowlist, h.transformableDenylist))
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
