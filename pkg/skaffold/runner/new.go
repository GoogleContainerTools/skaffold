/*
Copyright 2021 The Skaffold Authors

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

package runner

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/server"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/trigger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/verify"
)

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(ctx context.Context, runCtx *runcontext.RunContext) (*SkaffoldRunner, error) {
	event.InitializeState(runCtx)
	event.LogMetaEvent()
	eventV2.InitializeState(runCtx)
	eventV2.LogMetaEvent()
	_, endTrace := instrumentation.StartTrace(context.Background(), "NewForConfig")
	defer endTrace()

	tagger, err := tag.NewTaggerMux(runCtx)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating tagger: %w", err)
	}

	store := build.NewArtifactStore()
	g := graph.ToArtifactGraph(runCtx.Artifacts())
	sourceDependencies := graph.NewSourceDependenciesCache(runCtx, store, g)

	isLocalImage := func(imageName string) (bool, error) {
		return isImageLocal(runCtx, imageName)
	}

	// Always add skaffold-specific labels, except during `skaffold render`
	labeller := label.NewLabeller(runCtx.AddSkaffoldLabels(), runCtx.CustomLabels(), runCtx.GetRunID())
	tester, err := getTester(ctx, runCtx, isLocalImage)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating tester: %w", err)
	}

	var deployer deploy.Deployer

	hydrationDir, err := util.GetHydrationDir(runCtx.Opts, runCtx.WorkingDir, true, isKptRendererOrDeployerUsed(runCtx.Pipelines))

	if err != nil {
		return nil, fmt.Errorf("getting render output path: %w", err)
	}

	renderer, err := GetRenderer(ctx, runCtx, hydrationDir, labeller.Labels(), runCtx.UsingLegacyHelmDeploy())
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	deployer, err = GetDeployer(ctx, runCtx, labeller, hydrationDir, runCtx.UsingLegacyHelmDeploy())
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating deployer: %w", err)
	}
	rOpts := platform.ResolverOpts{
		KubeContext:               runCtx.KubeContext,
		CliPlatformsSelection:     runCtx.Opts.Platforms,
		CheckClusterNodePlatforms: runCtx.CheckClusterNodePlatforms(),
		DisableMultiPlatformBuild: runCtx.DisableMultiPlatformBuild(),
	}

	platforms, err := platform.NewResolver(ctx, runCtx.Pipelines.All(), rOpts)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("getting target platforms: %w", err)
	}

	var verifier verify.Verifier
	verifier, err = GetVerifier(ctx, runCtx, labeller)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating verifier: %w", err)
	}

	depLister := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
		ctx, endTrace := instrumentation.StartTrace(ctx, "NewForConfig_depLister")
		defer endTrace()

		buildDependencies, err := sourceDependencies.SingleArtifactDependencies(ctx, artifact)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}

		testDependencies, err := tester.TestDependencies(ctx, artifact)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		return append(buildDependencies, testDependencies...), nil
	}

	artifactCache, err := cache.NewCache(ctx, runCtx, isLocalImage, depLister, g, store)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("initializing cache: %w", err)
	}
	// The Builder must be instantiated AFTER the Deployer, because the Deploy target influences
	// the Cluster object on the RunContext, which in turn influences whether or not we will push images.
	var builder build.Builder
	builder, err = build.NewBuilderMux(runCtx, store, artifactCache, func(p latest.Pipeline) (build.PipelineBuilder, error) {
		return GetBuilder(ctx, runCtx, store, sourceDependencies, p)
	})
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating builder: %w", err)
	}

	builder, tester, renderer, deployer = WithTimings(builder, tester, renderer, deployer, runCtx.CacheArtifacts())
	if runCtx.Notification() {
		deployer = WithNotification(deployer)
	}

	monitor := filemon.NewMonitor()
	intents, intentChan := setupIntents(runCtx)
	rtrigger, err := trigger.NewTrigger(runCtx, intents.IsAnyAutoEnabled)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating watch trigger: %w", err)
	}

	rbuilder := NewBuilder(builder, tagger, platforms, artifactCache, runCtx)
	return &SkaffoldRunner{
		Builder:            *rbuilder,
		Pruner:             Pruner{Builder: builder},
		renderer:           renderer,
		tester:             tester,
		deployer:           deployer,
		platforms:          platforms,
		monitor:            monitor,
		listener:           NewSkaffoldListener(monitor, rtrigger, sourceDependencies, intentChan),
		artifactStore:      store,
		sourceDependencies: sourceDependencies,
		labeller:           labeller,
		cache:              artifactCache,
		runCtx:             runCtx,
		intents:            intents,
		isLocalImage:       isLocalImage,
		verifier:           verifier,
	}, nil
}

func setupIntents(runCtx *runcontext.RunContext) (*Intents, chan bool) {
	intents := NewIntents(runCtx.AutoBuild(), runCtx.AutoSync(), runCtx.AutoDeploy())

	intentChan := make(chan bool, 1)
	setupTrigger("build", intents.SetBuild, intents.SetAutoBuild, intents.GetAutoBuild, server.SetBuildCallback, server.SetAutoBuildCallback, intentChan)
	setupTrigger("sync", intents.SetSync, intents.SetAutoSync, intents.GetAutoSync, server.SetSyncCallback, server.SetAutoSyncCallback, intentChan)
	setupTrigger("deploy", intents.SetDeploy, intents.SetAutoDeploy, intents.GetAutoDeploy, server.SetDeployCallback, server.SetAutoDeployCallback, intentChan)
	// Setup callback function to buildCallback since build is the start of the devloop.
	setupTrigger("devloop", intents.SetDevloop, intents.SetAutoDevloop, intents.GetAutoDevloop, server.SetDevloopCallback, server.SetAutoDevloopCallback, intentChan)

	return intents, intentChan
}

func setupTrigger(triggerName string, setIntent func(bool), setAutoTrigger func(bool), getAutoTrigger func() bool, singleTriggerCallback func(func()), autoTriggerCallback func(func(bool)), c chan<- bool) {
	setIntent(getAutoTrigger())
	// give the server a callback to set the intent value when a user request is received
	singleTriggerCallback(func() {
		if !getAutoTrigger() { // if auto trigger is disabled, we're in manual mode
			log.Entry(context.TODO()).Debugf("%s intent received, calling back to runner", triggerName)
			c <- true
			setIntent(true)
		}
	})

	// give the server a callback to update auto trigger value when a user request is received
	autoTriggerCallback(func(val bool) {
		log.Entry(context.TODO()).Debugf("%s auto trigger update to %t received, calling back to runner", triggerName, val)
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

	// if we're deploying to local Docker, all images must be local
	if pipeline.Deploy.DockerDeploy != nil {
		return true, nil
	}

	cl := runCtx.GetCluster()
	var pushImages bool

	switch {
	case runCtx.Opts.PushImages.Value() != nil:
		log.Entry(context.TODO()).Debugf("push value set via skaffold build --push flag, --push=%t", *runCtx.Opts.PushImages.Value())
		pushImages = *runCtx.Opts.PushImages.Value()
	case pipeline.Build.LocalBuild.Push == nil:
		pushImages = cl.PushImages
		log.Entry(context.TODO()).Debugf("push value not present in isImageLocal(), defaulting to %t because cluster.PushImages is %t", pushImages, cl.PushImages)
	default:
		pushImages = *pipeline.Build.LocalBuild.Push
	}
	return !pushImages, nil
}

func getTester(ctx context.Context, cfg test.Config, isLocalImage func(imageName string) (bool, error)) (test.Tester, error) {
	tester, err := test.NewTester(ctx, cfg, isLocalImage)
	if err != nil {
		return nil, err
	}

	return tester, nil
}

func isKptRendererOrDeployerUsed(pipelines runcontext.Pipelines) bool {
	for _, configName := range pipelines.AllOrderedConfigNames() {
		pipeline := pipelines.GetForConfigName(configName)
		renderConfig := pipeline.Render
		deployConfig := pipeline.Deploy

		if renderConfig.Kpt != nil {
			return true
		}

		if deployConfig.KptDeploy != nil {
			return true
		}
	}

	return false
}
