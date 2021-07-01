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
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
)

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(runCtx *runcontext.RunContext) (*SkaffoldRunner, error) {
	event.InitializeState(runCtx)
	event.LogMetaEvent()
	eventV2.InitializeState(runCtx)
	eventV2.LogMetaEvent()
	kubectlCLI := pkgkubectl.NewCLI(runCtx, "")
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

	var builder build.Builder
	builder, err = build.NewBuilderMux(runCtx, store, func(p latestV1.Pipeline) (build.PipelineBuilder, error) {
		return runner.GetBuilder(runCtx, store, sourceDependencies, p)
	})
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating builder: %w", err)
	}
	isLocalImage := func(imageName string) (bool, error) {
		return isImageLocal(runCtx, imageName)
	}
	labeller := label.NewLabeller(runCtx.AddSkaffoldLabels(), runCtx.CustomLabels(), runCtx.GetRunID())
	tester, err := getTester(runCtx, isLocalImage)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating tester: %w", err)
	}

	var deployer deploy.Deployer
	provider := deploy.ComponentProvider{
		Accessor:    access.NewAccessorProvider(labeller),
		Debugger:    debug.NewDebugProvider(runCtx),
		ImageLoader: loader.NewImageLoaderProvider(runCtx, kubectlCLI),
		Logger:      log.NewLogProvider(runCtx, kubectlCLI),
		Monitor:     status.NewMonitorProvider(labeller),
		Syncer:      sync.NewSyncProvider(runCtx, kubectlCLI),
	}

	deployer, err = runner.GetDeployer(runCtx, provider, labeller.Labels())
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating deployer: %w", err)
	}

	depLister := func(ctx context.Context, artifact *latestV1.Artifact) ([]string, error) {
		ctx, endTrace := instrumentation.StartTrace(ctx, "NewForConfig_depLister")
		defer endTrace()

		buildDependencies, err := sourceDependencies.SingleArtifactDependencies(ctx, artifact)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}

		testDependencies, err := tester.TestDependencies(artifact)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		return append(buildDependencies, testDependencies...), nil
	}

	artifactCache, err := cache.NewCache(runCtx, isLocalImage, depLister, g, store)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
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
		endTrace(instrumentation.TraceEndError(err))
		return nil, fmt.Errorf("creating watch trigger: %w", err)
	}

	rbuilder := runner.NewBuilder(builder, tagger, artifactCache, runCtx)
	return &SkaffoldRunner{
		Builder:            *rbuilder,
		Pruner:             runner.Pruner{Builder: builder},
		Tester:             tester,
		deployer:           deployer,
		monitor:            monitor,
		listener:           runner.NewSkaffoldListener(monitor, rtrigger, sourceDependencies, intentChan),
		artifactStore:      store,
		sourceDependencies: sourceDependencies,
		kubectlCLI:         kubectlCLI,
		labeller:           labeller,
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
