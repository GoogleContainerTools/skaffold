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

package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/segmentio/textio"
	"go.opentelemetry.io/otel/trace"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/component/kubernetes"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	k8slogger "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	kstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	renderutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

// Deployer deploys workflows using kubectl CLI.
type Deployer struct {
	*latest.KubectlDeploy

	accessor           access.Accessor
	imageLoader        loader.ImageLoader
	logger             k8slogger.Logger
	debugger           debug.Debugger
	statusMonitor      kstatus.Monitor
	syncer             sync.Syncer
	hookRunner         hooks.Runner
	originalImages     []graph.Artifact // the set of images marked as "local" by the Runner
	localImages        []graph.Artifact // the set of images parsed from the Deployer's manifest set
	podSelector        *kubernetes.ImageList
	hydratedManifests  []string
	workingDir         string
	globalConfig       string
	gcsManifestDir     string
	defaultRepo        *string
	multiLevelRepo     *bool
	kubectl            CLI
	insecureRegistries map[string]bool
	labeller           *label.DefaultLabeller
	skipRender         bool
	hydrationDir       string
	namespaces         *[]string

	transformableAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformableDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

// NewDeployer returns a new Deployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewDeployer(cfg Config, labeller *label.DefaultLabeller, d *latest.KubectlDeploy, hydrationDir string) (*Deployer, error) {
	defaultNamespace := ""
	if d.DefaultNamespace != nil {
		var err error
		defaultNamespace, err = util.ExpandEnvTemplate(*d.DefaultNamespace, nil)
		if err != nil {
			return nil, err
		}
	}

	podSelector := kubernetes.NewImageList()
	kubectl := NewCLI(cfg, d.Flags, defaultNamespace)
	namespaces, err := deployutil.GetAllPodNamespaces(cfg.GetNamespace(), cfg.GetPipelines())
	if err != nil {
		olog.Entry(context.TODO()).Warn("unable to parse namespaces - deploy might not work correctly!")
	}
	logger := component.NewLogger(cfg, kubectl.CLI, podSelector, &namespaces)
	transformableAllowlist, transformableDenylist, err := renderutil.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return nil, err
	}
	return &Deployer{
		KubectlDeploy:      d,
		podSelector:        podSelector,
		namespaces:         &namespaces,
		accessor:           component.NewAccessor(cfg, cfg.GetKubeContext(), kubectl.CLI, podSelector, labeller, &namespaces),
		debugger:           component.NewDebugger(cfg.Mode(), podSelector, &namespaces, cfg.GetKubeContext()),
		imageLoader:        component.NewImageLoader(cfg, kubectl.CLI),
		logger:             logger,
		statusMonitor:      component.NewMonitor(cfg, cfg.GetKubeContext(), labeller, &namespaces),
		syncer:             component.NewSyncer(kubectl.CLI, &namespaces, logger.GetFormatter()),
		hookRunner:         hooks.NewDeployRunner(kubectl.CLI, d.LifecycleHooks, &namespaces, logger.GetFormatter(), hooks.NewDeployEnvOpts(labeller.GetRunID(), kubectl.KubeContext, namespaces)),
		workingDir:         cfg.GetWorkingDir(),
		globalConfig:       cfg.GlobalConfig(),
		defaultRepo:        cfg.DefaultRepo(),
		multiLevelRepo:     cfg.MultiLevelRepo(),
		kubectl:            kubectl,
		insecureRegistries: cfg.GetInsecureRegistries(),
		skipRender:         cfg.SkipRender(),
		labeller:           labeller,
		// hydratedManifests refers to the DIR in the `skaffold apply DIR`. Used in both v1 and v2.
		hydratedManifests: cfg.HydratedManifests(),
		// hydrationDir refers to the path where the hydrated manifests are stored, this is introduced in v2.
		hydrationDir: hydrationDir,

		transformableAllowlist: transformableAllowlist,
		transformableDenylist:  transformableDenylist,
	}, nil
}

func (k *Deployer) GetAccessor() access.Accessor {
	return k.accessor
}

func (k *Deployer) GetDebugger() debug.Debugger {
	return k.debugger
}

func (k *Deployer) GetLogger() log.Logger {
	return k.logger
}

func (k *Deployer) GetStatusMonitor() status.Monitor {
	return k.statusMonitor
}

func (k *Deployer) GetSyncer() sync.Syncer {
	return k.syncer
}

func (k *Deployer) RegisterLocalImages(images []graph.Artifact) {
	k.localImages = images
}

func (k *Deployer) TrackBuildArtifacts(artifacts []graph.Artifact) {
	deployutil.AddTagsToPodSelector(artifacts, k.podSelector)
	k.logger.RegisterArtifacts(artifacts)
}

func (k *Deployer) trackNamespaces(namespaces []string) {
	*k.namespaces = deployutil.ConsolidateNamespaces(*k.namespaces, namespaces)
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact, manifests manifest.ManifestList) error {
	var (
		err      error
		childCtx context.Context
		endTrace func(...trace.SpanOption)
	)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})

	// Check that the cluster is reachable.
	// This gives a better error message when the cluster can't
	// be reached.
	if err := kubernetes.FailIfClusterIsNotReachable(k.kubectl.KubeContext); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	// if any hydrated manifests are passed to `skaffold apply`, only deploy these
	// also, manually set the labels to ensure the runID is added
	if len(k.hydratedManifests) > 0 {
		_, endTrace = instrumentation.StartTrace(ctx, "Deploy_readHydratedManifests")
		manifests, err = k.kubectl.ReadManifests(ctx, k.hydratedManifests)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		manifests, err = manifests.SetLabels(k.labeller.Labels(), manifest.NewResourceSelectorLabels(k.transformableAllowlist, k.transformableDenylist))
		endTrace()
	} else if k.skipRender {
		childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_readManifests")
		manifests, err = k.readManifests(childCtx, false)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		manifests, err = manifests.SetLabels(k.labeller.Labels(), manifest.NewResourceSelectorLabels(k.transformableAllowlist, k.transformableDenylist))
		endTrace()
	}

	if err != nil {
		return err
	}

	if len(manifests) == 0 {
		return fmt.Errorf("nothing to deploy")
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Deploy_LoadImages")
	if err := k.imageLoader.LoadImages(childCtx, out, k.localImages, k.originalImages, builds); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Deploy_CollectNamespaces")
	namespaces, err := manifests.CollectNamespaces()
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not fetch deployed resource namespace. "+
			"This might cause port-forward and deploy health-check to fail: %w", err))
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_WaitForDeletions")
	if err := k.kubectl.WaitForDeletions(childCtx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_KubectlApply")
	if err := k.kubectl.Apply(childCtx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	k.TrackBuildArtifacts(builds)
	k.statusMonitor.RegisterDeployManifests(manifests)
	endTrace()
	k.trackNamespaces(namespaces)
	return nil
}

func (k *Deployer) HasRunnableHooks() bool {
	return len(k.KubectlDeploy.LifecycleHooks.PreHooks) > 0 || len(k.KubectlDeploy.LifecycleHooks.PostHooks) > 0
}

func (k *Deployer) PreDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PreHooks")
	if err := k.hookRunner.RunPreHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (k *Deployer) PostDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PostHooks")
	if err := k.hookRunner.RunPostHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (k *Deployer) manifestFiles(manifests []string) ([]string, error) {
	var nonURLManifests, gcsManifests []string
	for _, manifest := range manifests {
		switch {
		case util.IsURL(manifest):
		case strings.HasPrefix(manifest, "gs://"):
			gcsManifests = append(gcsManifests, manifest)
		default:
			nonURLManifests = append(nonURLManifests, manifest)
		}
	}

	list, err := util.ExpandPathsGlob(k.workingDir, nonURLManifests)
	if err != nil {
		return nil, userErr(fmt.Errorf("expanding kubectl manifest paths: %w", err))
	}

	if len(gcsManifests) != 0 {
		// return tmp dir of the downloaded manifests
		tmpDir, err := manifest.DownloadFromGCS(gcsManifests)
		if err != nil {
			return nil, userErr(fmt.Errorf("downloading from GCS: %w", err))
		}
		k.gcsManifestDir = tmpDir
		l, err := util.ExpandPathsGlob(tmpDir, []string{"*"})
		if err != nil {
			return nil, userErr(fmt.Errorf("expanding kubectl manifest paths: %w", err))
		}
		list = append(list, l...)
	}

	var filteredManifests []string
	for _, f := range list {
		if !kubernetes.HasKubernetesFileExtension(f) {
			if !stringslice.Contains(manifests, f) {
				olog.Entry(context.TODO()).Infof("refusing to deploy/delete non {json, yaml} file %s", f)
				olog.Entry(context.TODO()).Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
				continue
			}
		}
		filteredManifests = append(filteredManifests, f)
	}

	return filteredManifests, nil
}

// readManifests reads the manifests to deploy/delete.
func (k *Deployer) readManifests(ctx context.Context, offline bool) (manifest.ManifestList, error) {
	var manifests []string
	var err error

	// v1 kubectl deployer is used. No manifest hydration.
	if len(k.KubectlDeploy.Manifests) > 0 {
		olog.Entry(ctx).Warnln("`deploy.kubectl.manifests` (DEPRECATED) are given, skaffold will skip the `manifests` field. " +
			"If you expect skaffold to render the resources from the `manifests`, please delete the `deploy.kubectl.manifests` field.")
		manifests, err = k.Dependencies()
		if err != nil {
			return nil, listManifestErr(fmt.Errorf("listing manifests: %w", err))
		}
	} else {
		// v2 kubectl deployer is used. The manifests are read from the hydrated directory.
		manifests, err = k.manifestFiles([]string{filepath.Join(k.hydrationDir, "*")})
		if err != nil {
			return nil, listManifestErr(fmt.Errorf("listing manifests: %w", err))
		}
	}

	// Clean the temporary directory that holds the manifests downloaded from GCS
	defer os.RemoveAll(k.gcsManifestDir)

	// Append URL manifests. URL manifests are excluded from `Dependencies`.
	hasURLManifest := false
	for _, manifest := range k.KubectlDeploy.Manifests {
		if util.IsURL(manifest) {
			manifests = append(manifests, manifest)
			hasURLManifest = true
		}
	}

	if len(manifests) == 0 {
		return manifest.ManifestList{}, nil
	}

	if !offline {
		return k.kubectl.ReadManifests(ctx, manifests)
	}

	// In case no URLs are provided, we can stay offline - no need to run "kubectl create" which
	// would try to connect to a cluster (https://github.com/kubernetes/kubernetes/issues/51475)
	if hasURLManifest {
		return nil, offlineModeErr()
	}
	return createManifestList(manifests)
}

func createManifestList(manifests []string) (manifest.ManifestList, error) {
	var manifestList manifest.ManifestList
	for _, manifestFilePath := range manifests {
		manifestFileContent, err := ioutil.ReadFile(manifestFilePath)
		if err != nil {
			return nil, readManifestErr(fmt.Errorf("reading manifest file %v: %w", manifestFilePath, err))
		}
		manifestList.Append(manifestFileContent)
	}
	return manifestList, nil
}

// readRemoteManifests will try to read manifests from the given kubernetes
// context in the specified namespace and for the specified type
func (k *Deployer) readRemoteManifest(ctx context.Context, name string) ([]byte, error) {
	var args []string
	ns := ""
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		ns = parts[0]
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := k.kubectl.RunInNamespace(ctx, nil, &manifest, "get", ns, args...)
	if err != nil {
		return nil, readRemoteManifestErr(fmt.Errorf("getting remote manifests: %w", err))
	}

	return manifest.Bytes(), nil
}

func (k *Deployer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Render_renderManifests")
	manifests, err := k.renderManifests(childCtx, out, builds, offline)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	k.statusMonitor.RegisterDeployManifests(manifests)
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Render_manifest.Write")
	defer endTrace()
	return manifest.Write(manifests.String(), filepath, out)
}

// renderManifests transforms the manifests' images with the actual image sha1 built from skaffold build.
func (k *Deployer) renderManifests(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestList, error) {
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		output.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
		output.Default.Fprintln(out, err)
	}

	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, deployerr.DebugHelperRetrieveErr(fmt.Errorf("retrieving debug helpers registry: %w", err))
	}
	var localManifests, remoteManifests manifest.ManifestList
	localManifests, err = k.readManifests(ctx, offline)
	if err != nil {
		return nil, err
	}

	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(ctx, m)
		if err != nil {
			return nil, err
		}

		remoteManifests = append(remoteManifests, manifest)
	}

	originalManifests := append(localManifests, remoteManifests...)

	if len(k.originalImages) == 0 {
		// TODO(aaron-prindle) maybe use different resoureselector?
		k.originalImages, err = originalManifests.GetImages(manifest.NewResourceSelectorImages(k.transformableAllowlist, k.transformableDenylist))
		// k.originalImages, err = originalManifests.GetImages(k.transformableAllowlist, k.transformableDenylist)
		if err != nil {
			return nil, err
		}
	}

	if len(originalManifests) == 0 {
		return nil, nil
	}

	if len(builds) == 0 {
		for _, artifact := range k.originalImages {
			tag, err := deployutil.ApplyDefaultRepo(k.globalConfig, k.defaultRepo, artifact.Tag)
			if err != nil {
				return nil, err
			}
			builds = append(builds, graph.Artifact{
				ImageName: artifact.ImageName,
				Tag:       tag,
			})
		}
	}
	if len(remoteManifests) > 0 {
		remoteManifests, err = remoteManifests.ReplaceRemoteManifestImages(ctx, builds, manifest.NewResourceSelectorImages(k.transformableAllowlist, k.transformableDenylist))
		if err != nil {
			return nil, err
		}
	}
	if len(localManifests) > 0 {
		localManifests, err = localManifests.ReplaceImages(ctx, builds, manifest.NewResourceSelectorImages(k.transformableAllowlist, k.transformableDenylist))
		if err != nil {
			return nil, err
		}
	}

	modifiedManifests := append(localManifests, remoteManifests...)

	if modifiedManifests, err = manifest.ApplyTransforms(modifiedManifests, builds, k.insecureRegistries, debugHelpersRegistry); err != nil {
		return nil, err
	}

	return modifiedManifests.SetLabels(k.labeller.Labels(), manifest.NewResourceSelectorLabels(k.transformableAllowlist, k.transformableDenylist))
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})
	manifests, err := k.readManifests(ctx, false)
	if err != nil {
		return err
	}
	if dryRun {
		for _, manifest := range manifests {
			output.White.Fprintf(out, "---\n%s", manifest)
		}
		return nil
	}
	// revert remote manifests
	// TODO(dgageot): That seems super dangerous and I don't understand
	// why we need to update resources just before we delete them.
	if len(k.RemoteManifests) > 0 {
		var rm manifest.ManifestList
		for _, m := range k.RemoteManifests {
			manifest, err := k.readRemoteManifest(ctx, m)
			if err != nil {
				return err
			}
			rm = append(rm, manifest)
		}

		upd, err := rm.ReplaceRemoteManifestImages(ctx, k.originalImages, manifest.NewResourceSelectorImages(k.transformableAllowlist, k.transformableDenylist))
		if err != nil {
			return err
		}

		if err := k.kubectl.Apply(ctx, out, upd); err != nil {
			return err
		}
	}

	if err := k.kubectl.Delete(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return err
	}

	return nil
}

// Dependencies lists all the files that describe what needs to be deployed.
func (k *Deployer) Dependencies() ([]string, error) {
	return k.manifestFiles(k.KubectlDeploy.Manifests)
}
