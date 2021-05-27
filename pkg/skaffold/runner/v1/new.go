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

package v1

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(runCtx *runcontext.RunContext) (*SkaffoldRunner, error) {
	event.InitializeState(runCtx)
	event.LogMetaEvent()
	kubectlCLI := pkgkubectl.NewCLI(runCtx, "")

	tagger, err := tag.NewTaggerMux(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating tagger: %w", err)
	}

	store := build.NewArtifactStore()
	g := graph.ToArtifactGraph(runCtx.Artifacts())
	sourceDependencies := graph.NewSourceDependenciesCache(runCtx, store, g)

	var builder build.Builder
	builder, err = build.NewBuilderMux(runCtx, store, func(p latestV1.Pipeline) (build.PipelineBuilder, error) {
		return runner.GetBuilder(runCtx, store, sourceDependencies, p)
	})
	if err != nil {
		return nil, fmt.Errorf("creating builder: %w", err)
	}
	isLocalImage := func(imageName string) (bool, error) {
		return isImageLocal(runCtx, imageName)
	}
	labeller := label.NewLabeller(runCtx.AddSkaffoldLabels(), runCtx.CustomLabels())
	tester, err := getTester(runCtx, isLocalImage)
	if err != nil {
		return nil, fmt.Errorf("creating tester: %w", err)
	}
	syncer := getSyncer(runCtx)
	var deployer deploy.Deployer
	deployer, err = getDeployer(runCtx, labeller.Labels())
	if err != nil {
		return nil, fmt.Errorf("creating deployer: %w", err)
	}

	depLister := func(ctx context.Context, artifact *latestV1.Artifact) ([]string, error) {
		buildDependencies, err := sourceDependencies.SingleArtifactDependencies(ctx, artifact)
		if err != nil {
			return nil, err
		}

		testDependencies, err := tester.TestDependencies(artifact)
		if err != nil {
			return nil, err
		}

		return append(buildDependencies, testDependencies...), nil
	}

	artifactCache, err := cache.NewCache(runCtx, isLocalImage, depLister, g, store)
	if err != nil {
		return nil, fmt.Errorf("initializing cache: %w", err)
	}

	builder, tester, deployer = runner.WithTimings(builder, tester, deployer, runCtx.CacheArtifacts())
	if runCtx.Notification() {
		deployer = runner.WithNotification(deployer)
	}

	monitor := filemon.NewMonitor()
	intents, intentChan := setupIntents(runCtx)
	rtrigger, err := trigger.NewTrigger(runCtx, intents.IsAnyAutoEnabled)
	if err != nil {
		return nil, fmt.Errorf("creating watch trigger: %w", err)
	}

	podSelectors := kubernetes.NewImageList()

	rbuilder := runner.NewBuilder(builder, tagger, artifactCache, podSelectors, runCtx)
	return &SkaffoldRunner{
		Builder:            *rbuilder,
		Pruner:             runner.Pruner{Builder: builder},
		Tester:             tester,
		deployer:           deployer,
		syncer:             syncer,
		monitor:            monitor,
		listener:           runner.NewSkaffoldListener(monitor, rtrigger, sourceDependencies, intentChan),
		artifactStore:      store,
		sourceDependencies: sourceDependencies,
		kubectlCLI:         kubectlCLI,
		labeller:           labeller,
		podSelector:        podSelectors,
		cache:              artifactCache,
		runCtx:             runCtx,
		intents:            intents,
		isLocalImage:       isLocalImage,
	}, nil
}

func setupIntents(runCtx *runcontext.RunContext) (*runner.Intents, chan bool) {
	intents := runner.NewIntents(runCtx.AutoBuild(), runCtx.AutoSync(), runCtx.AutoDeploy())

	intentChan := make(chan bool, 1)
	setupTrigger("build", intents.SetBuild, intents.SetAutoBuild, intents.GetAutoBuild, server.SetBuildCallback, server.SetAutoBuildCallback, intentChan)
	setupTrigger("sync", intents.SetSync, intents.SetAutoSync, intents.GetAutoSync, server.SetSyncCallback, server.SetAutoSyncCallback, intentChan)
	setupTrigger("deploy", intents.SetDeploy, intents.SetAutoDeploy, intents.GetAutoDeploy, server.SetDeployCallback, server.SetAutoDeployCallback, intentChan)

	return intents, intentChan
}

func setupTrigger(triggerName string, setIntent func(bool), setAutoTrigger func(bool), getAutoTrigger func() bool, singleTriggerCallback func(func()), autoTriggerCallback func(func(bool)), c chan<- bool) {
	setIntent(getAutoTrigger())
	// give the server a callback to set the intent value when a user request is received
	singleTriggerCallback(func() {
		if !getAutoTrigger() { // if auto trigger is disabled, we're in manual mode
			logrus.Debugf("%s intent received, calling back to runner", triggerName)
			c <- true
			setIntent(true)
		}
	})

	// give the server a callback to update auto trigger value when a user request is received
	autoTriggerCallback(func(val bool) {
		logrus.Debugf("%s auto trigger update to %t received, calling back to runner", triggerName, val)
		// signal chan only when auto trigger is set to true
		if val {
			c <- true
		}
		setAutoTrigger(val)
		setIntent(val)
	})
}

func isImageLocal(runCtx *runcontext.RunContext, imageName string) (bool, error) {
	pipeline, found := runCtx.PipelineForImage(imageName)
	if !found {
		pipeline = runCtx.DefaultPipeline()
	}
	if pipeline.Build.GoogleCloudBuild != nil || pipeline.Build.Cluster != nil {
		return false, nil
	}

	cl := runCtx.GetCluster()
	var pushImages bool

	switch {
	case runCtx.Opts.PushImages.Value() != nil:
		logrus.Debugf("push value set via skaffold build --push flag, --push=%t", *runCtx.Opts.PushImages.Value())
		pushImages = *runCtx.Opts.PushImages.Value()
	case pipeline.Build.LocalBuild.Push == nil:
		pushImages = cl.PushImages
		logrus.Debugf("push value not present in isImageLocal(), defaulting to %t because cluster.PushImages is %t", pushImages, cl.PushImages)
	default:
		pushImages = *pipeline.Build.LocalBuild.Push
	}
	return !pushImages, nil
}

func getTester(cfg test.Config, isLocalImage func(imageName string) (bool, error)) (test.Tester, error) {
	tester, err := test.NewTester(cfg, isLocalImage)
	if err != nil {
		return nil, err
	}

	return tester, nil
}

func getSyncer(cfg sync.Config) sync.Syncer {
	return sync.NewSyncer(cfg)
}

/*
The "default deployer" is used in `skaffold apply`, which uses a `kubectl` deployer to actuate resources
on a cluster regardless of provided deployer configuration in the skaffold.yaml.
The default deployer will honor a select set of deploy configuration from an existing skaffold.yaml:
	- deploy.StatusCheckDeadlineSeconds
	- deploy.Logs.Prefix
	- deploy.Kubectl.Flags
	- deploy.Kubectl.DefaultNamespace
	- deploy.Kustomize.Flags
	- deploy.Kustomize.DefaultNamespace
For a multi-config project, we do not currently support resolving conflicts between differing sets of this deploy configuration.
Therefore, in this function we do implicit validation of the provided configuration, and fail if any conflict cannot be resolved.
*/
func getDefaultDeployer(runCtx *runcontext.RunContext, labels map[string]string) (deploy.Deployer, error) {
	deployCfgs := runCtx.DeployConfigs()

	var kFlags *latestV1.KubectlFlags
	var logPrefix string
	var defaultNamespace *string
	var kubeContext string
	statusCheckTimeout := -1

	for _, d := range deployCfgs {
		if d.KubeContext != "" {
			if kubeContext != "" && kubeContext != d.KubeContext {
				return nil, errors.New("cannot resolve active Kubernetes context - multiple contexts configured in skaffold.yaml")
			}
			kubeContext = d.KubeContext
		}
		if d.StatusCheckDeadlineSeconds != 0 && d.StatusCheckDeadlineSeconds != int(status.DefaultStatusCheckDeadline.Seconds()) {
			if statusCheckTimeout != -1 && statusCheckTimeout != d.StatusCheckDeadlineSeconds {
				return nil, fmt.Errorf("found multiple status check timeouts in skaffold.yaml (not supported in `skaffold apply`): %d, %d", statusCheckTimeout, d.StatusCheckDeadlineSeconds)
			}
			statusCheckTimeout = d.StatusCheckDeadlineSeconds
		}
		if d.Logs.Prefix != "" {
			if logPrefix != "" && logPrefix != d.Logs.Prefix {
				return nil, fmt.Errorf("found multiple log prefixes in skaffold.yaml (not supported in `skaffold apply`): %s, %s", logPrefix, d.Logs.Prefix)
			}
			logPrefix = d.Logs.Prefix
		}
		var currentDefaultNamespace *string
		var currentKubectlFlags latestV1.KubectlFlags
		if d.KubectlDeploy != nil {
			currentDefaultNamespace = d.KubectlDeploy.DefaultNamespace
			currentKubectlFlags = d.KubectlDeploy.Flags
		}
		if d.KustomizeDeploy != nil {
			currentDefaultNamespace = d.KustomizeDeploy.DefaultNamespace
			currentKubectlFlags = d.KustomizeDeploy.Flags
		}
		if kFlags == nil {
			kFlags = &currentKubectlFlags
		}
		if err := validateKubectlFlags(kFlags, currentKubectlFlags); err != nil {
			return nil, err
		}
		if currentDefaultNamespace != nil {
			if defaultNamespace != nil && *defaultNamespace != *currentDefaultNamespace {
				return nil, fmt.Errorf("found multiple namespaces in skaffold.yaml (not supported in `skaffold apply`): %s, %s", *defaultNamespace, *currentDefaultNamespace)
			}
			defaultNamespace = currentDefaultNamespace
		}
	}
	if kFlags == nil {
		kFlags = &latestV1.KubectlFlags{}
	}
	k := &latestV1.KubectlDeploy{
		Flags:            *kFlags,
		DefaultNamespace: defaultNamespace,
	}
	defaultDeployer, err := kubectl.NewDeployer(runCtx, labels, k)
	if err != nil {
		return nil, fmt.Errorf("instantiating default kubectl deployer: %w", err)
	}
	return defaultDeployer, nil
}

func validateKubectlFlags(flags *latestV1.KubectlFlags, additional latestV1.KubectlFlags) error {
	errStr := "conflicting sets of kubectl deploy flags not supported in `skaffold apply` (flag: %s)"
	if additional.DisableValidation != flags.DisableValidation {
		return fmt.Errorf(errStr, strconv.FormatBool(additional.DisableValidation))
	}
	for _, flag := range additional.Apply {
		if !util.StrSliceContains(flags.Apply, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	for _, flag := range additional.Delete {
		if !util.StrSliceContains(flags.Delete, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	for _, flag := range additional.Global {
		if !util.StrSliceContains(flags.Global, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	return nil
}

func getDeployer(runCtx *runcontext.RunContext, labels map[string]string) (deploy.Deployer, error) {
	if runCtx.Opts.Apply {
		return getDefaultDeployer(runCtx, labels)
	}

	deployerCfg := runCtx.Deployers()

	var deployers deploy.DeployerMux
	for _, d := range deployerCfg {
		if d.HelmDeploy != nil {
			h, err := helm.NewDeployer(runCtx, labels, d.HelmDeploy)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, h)
		}

		if d.KptDeploy != nil {
			deployers = append(deployers, kpt.NewDeployer(runCtx, labels, d.KptDeploy))
		}

		if d.KubectlDeploy != nil {
			deployer, err := kubectl.NewDeployer(runCtx, labels, d.KubectlDeploy)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, deployer)
		}

		if d.KustomizeDeploy != nil {
			deployer, err := kustomize.NewDeployer(runCtx, labels, d.KustomizeDeploy)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, deployer)
		}
	}

	return deployers, nil
}
