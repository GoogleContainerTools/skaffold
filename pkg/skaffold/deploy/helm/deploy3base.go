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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/component/kubernetes"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kloader "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	kstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/blang/semver"
	backoff "github.com/cenkalti/backoff/v4"
)

var (
	// versionRegex extracts version from "helm version --client", for instance: "2.14.0-rc.2"
	versionRegex = regexp.MustCompile(`v?(\d[\w.\-]+)`)

	// helm3Version represents the version cut-off for helm3 behavior
	helm3Version  = semver.MustParse("3.0.0-beta.0")
	helm32Version = semver.MustParse("3.2.0")

	// helm31Version represents the version cut-off for helm3.1 post-renderer behavior
	helm31Version = semver.MustParse("3.1.0")

	// error to throw when helm version can't be determined
	versionErrorString = "failed to determine binary version: %w"

	// osExecutable allows for replacing the skaffold binary for testing purposes
	osExecutable = os.Executable
)

// Deployer3 deploys workflows using the helm CLI 3.0.0
type Deployer3 struct {
	*latest.HelmDeploy

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
	JSONParseConfig() latest.JSONParseConfig
}

// NewBase returns a configured Deployer3.
func NewBase(ctx context.Context, cfg Config, labeller *label.DefaultLabeller, h *latest.HelmDeploy, hv semver.Version) (*Deployer3, error) {
	originalImages := []graph.Artifact{}
	for _, release := range h.Releases {
		for _, v := range release.ArtifactOverrides {
			originalImages = append(originalImages, graph.Artifact{
				ImageName: v,
			})
		}
	}

	podSelector := kubernetes.NewImageList()
	kubectl := pkgkubectl.NewCLI(cfg, cfg.GetKubeNamespace())
	namespaces, err := deployutil.GetAllPodNamespaces(cfg.GetNamespace(), cfg.GetPipelines())
	if err != nil {
		olog.Entry(context.TODO()).Warn("unable to parse namespaces - deploy might not work correctly!")
	}
	logger := component.NewLogger(cfg, kubectl, podSelector, &namespaces)
	return &Deployer3{
		HelmDeploy:     h,
		podSelector:    podSelector,
		namespaces:     &namespaces,
		accessor:       component.NewAccessor(cfg, cfg.GetKubeContext(), kubectl, podSelector, labeller, &namespaces),
		debugger:       component.NewDebugger(cfg.Mode(), podSelector, &namespaces, cfg.GetKubeContext()),
		imageLoader:    component.NewImageLoader(cfg, kubectl),
		logger:         logger,
		statusMonitor:  component.NewMonitor(cfg, cfg.GetKubeContext(), labeller, &namespaces),
		syncer:         component.NewSyncer(kubectl, &namespaces, logger.GetFormatter()),
		hookRunner:     hooks.NewDeployRunner(kubectl, h.LifecycleHooks, &namespaces, logger.GetFormatter(), hooks.NewDeployEnvOpts(labeller.GetRunID(), kubectl.KubeContext, namespaces)),
		originalImages: originalImages,
		kubeContext:    cfg.GetKubeContext(),
		kubeConfig:     cfg.GetKubeConfig(),
		namespace:      cfg.GetKubeNamespace(),
		forceDeploy:    cfg.ForceDeploy(),
		configFile:     cfg.ConfigurationFile(),
		labels:         labeller.Labels(),
		bV:             hv,
		enableDebug:    cfg.Mode() == config.RunModes.Debug,
		isMultiConfig:  cfg.IsMultiConfig(),
	}, nil
}

func (h *Deployer3) trackNamespaces(namespaces []string) {
	*h.namespaces = deployutil.ConsolidateNamespaces(*h.namespaces, namespaces)
}

func (h *Deployer3) GetAccessor() access.Accessor {
	return h.accessor
}

func (h *Deployer3) GetDebugger() debug.Debugger {
	return h.debugger
}

func (h *Deployer3) GetLogger() log.Logger {
	return h.logger
}

func (h *Deployer3) GetStatusMonitor() status.Monitor {
	return h.statusMonitor
}

func (h *Deployer3) GetSyncer() sync.Syncer {
	return h.syncer
}

func (h *Deployer3) RegisterLocalImages(images []graph.Artifact) {
	h.localImages = images
}

func (h *Deployer3) TrackBuildArtifacts(artifacts []graph.Artifact) {
	deployutil.AddTagsToPodSelector(artifacts, h.originalImages, h.podSelector)
	h.logger.RegisterArtifacts(artifacts)
}

// Dependencies returns a list of files that the deployer depends on.
func (h *Deployer3) Dependencies() ([]string, error) {
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
func (h *Deployer3) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
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

func (h *Deployer3) HasRunnableHooks() bool {
	return len(h.HelmDeploy.LifecycleHooks.PreHooks) > 0 || len(h.HelmDeploy.LifecycleHooks.PostHooks) > 0
}

func (h *Deployer3) PreDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PreHooks")
	if err := h.hookRunner.RunPreHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (h *Deployer3) PostDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PostHooks")
	if err := h.hookRunner.RunPostHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

// getReleaseManifest confirms that a release is visible to helm and returns the release manifest
func (h *Deployer3) getReleaseManifest(ctx context.Context, releaseName string, namespace string) (bytes.Buffer, error) {
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

	return b, err
}

func (h *Deployer3) getInstallOpts(releaseName string, r latest.HelmRelease, helmVersion semver.Version, chartVersion string) installOpts {
	return installOpts{
		releaseName: releaseName,
		upgrade:     true,
		flags:       h.Flags.Upgrade,
		force:       h.forceDeploy,
		chartPath:   chartSource(r),
		helmVersion: helmVersion,
		repo:        r.Repo,
		version:     chartVersion,
	}
}

// packageChart packages the chart and returns the path to the resulting chart archive
func (h *Deployer3) packageChart(ctx context.Context, r latest.HelmRelease) (string, error) {
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

func chartSource(r latest.HelmRelease) string {
	if r.RemoteChart != "" {
		return r.RemoteChart
	}
	return r.ChartPath
}

func warnAboutUnusedImages(builds []graph.Artifact, valuesSet map[string]bool) {
	for _, b := range builds {
		if !valuesSet[b.Tag] {
			warnings.Printf("image [%s] is not used.", b.Tag)
			warnings.Printf("See helm documentation on how to replace image names with their actual tags: https://skaffold.dev/docs/pipeline-stages/deployers/helm/#image-configuration")
		}
	}
}
