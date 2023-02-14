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
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/segmentio/textio"
	"go.opentelemetry.io/otel/trace"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/component/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	k8slogger "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
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
)

// Deployer deploys workflows using kubectl CLI.
type Deployer struct {
	configName string

	*latest.KubectlDeploy

	accessor           access.Accessor
	imageLoader        loader.ImageLoader
	logger             k8slogger.Logger
	debugger           debug.Debugger
	statusMonitor      kstatus.Monitor
	syncer             sync.Syncer
	hookRunner         hooks.Runner
	originalImages     []graph.Artifact // the set of images parsed from the Deployer's manifest set
	localImages        []graph.Artifact // the set of images marked as "local" by the Runner
	podSelector        *kubernetes.ImageList
	hydratedManifests  []string
	workingDir         string
	globalConfig       string
	defaultRepo        *string
	multiLevelRepo     *bool
	kubectl            CLI
	insecureRegistries map[string]bool
	labeller           *label.DefaultLabeller
	namespaces         *[]string

	transformableAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformableDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

// NewDeployer returns a new Deployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewDeployer(cfg Config, labeller *label.DefaultLabeller, d *latest.KubectlDeploy, artifacts []*latest.Artifact, configName string) (*Deployer, error) {
	defaultNamespace := ""
	b, err := util.RunCmdOutOnce(context.TODO(), exec.Command("kubectl", "config", "view", "--minify", "-o", "jsonpath='{..namespace}'"))
	if err == nil {
		defaultNamespace = strings.Trim(string(b), "'")
		if defaultNamespace == "default" {
			defaultNamespace = ""
		}
	}
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

	var ogImages []graph.Artifact
	for _, artifact := range artifacts {
		ogImages = append(ogImages, graph.Artifact{
			ImageName:   artifact.ImageName,
			RuntimeType: artifact.RuntimeType,
		})
	}

	return &Deployer{
		originalImages:     ogImages,
		configName:         configName,
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
		labeller:           labeller,
		// hydratedManifests refers to the DIR in the `skaffold apply DIR`. Used in both v1 and v2.
		hydratedManifests: cfg.HydratedManifests(),
		// hydrationDir refers to the path where the hydrated manifests are stored, this is introduced in v2.

		transformableAllowlist: transformableAllowlist,
		transformableDenylist:  transformableDenylist,
	}, nil
}

func (k *Deployer) ConfigName() string {
	return k.configName
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

func (k *Deployer) TrackBuildArtifacts(builds, deployedImages []graph.Artifact) {
	deployutil.AddTagsToPodSelector(builds, deployedImages, k.podSelector)

	// This is to register color for each image logging with a round-robin way.
	k.logger.RegisterArtifacts(builds)
}

func (k *Deployer) trackNamespaces(namespaces []string) {
	*k.namespaces = deployutil.ConsolidateNamespaces(*k.namespaces, namespaces)
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact, manifestsByConfig manifest.ManifestListByConfig) error {
	manifests := manifestsByConfig.GetForConfig(k.ConfigName())
	var (
		err      error
		childCtx context.Context
		endTrace func(...trace.SpanEndOption)
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
	}

	if err != nil {
		return err
	}

	if len(manifests) == 0 {
		return fmt.Errorf("nothing to deploy")
	}

	// Add debug transformations
	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return err
	}
	if manifests, err = manifest.ApplyTransforms(manifests, builds, k.insecureRegistries, debugHelpersRegistry); err != nil {
		return err
	}

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_LoadImages")
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
	deployedImages, _ := manifests.GetImages(manifest.NewResourceSelectorImages(manifest.TransformAllowlist, manifest.TransformDenylist))

	k.TrackBuildArtifacts(builds, deployedImages)
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

// Cleanup deletes what was deployed by calling Deploy.
func (k *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool, manifestsByConfig manifest.ManifestListByConfig) error {
	var manifests manifest.ManifestList
	manifests = append(manifests, manifestsByConfig.GetForConfig(k.ConfigName())...)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})
	if dryRun {
		for _, manifestP := range manifests {
			output.White.Fprintf(out, "---\n%s", manifestP)
		}
		return nil
	}

	if err := k.kubectl.Delete(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return err
	}

	return nil
}

// Dependencies lists all the files that describe what needs to be deployed.
func (k *Deployer) Dependencies() ([]string, error) {
	return []string{}, nil
}
