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

package runner

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
)

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(runCtx *runcontext.RunContext) (*SkaffoldRunner, error) {
	event.InitializeState(runCtx.GetPipelines(), runCtx.GetKubeContext(), runCtx.AutoBuild(), runCtx.AutoDeploy(), runCtx.AutoSync())
	event.LogMetaEvent()
	kubectlCLI := pkgkubectl.NewCLI(runCtx, "")

	tagger, err := tag.NewTaggerMux(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating tagger: %w", err)
	}

	store := build.NewArtifactStore()
	var builder build.Builder
	builder, err = build.NewBuilderMux(runCtx, store, func(p latest.Pipeline) (build.PipelineBuilder, error) {
		return getBuilder(runCtx, store, p)
	})
	if err != nil {
		return nil, fmt.Errorf("creating builder: %w", err)
	}
	isLocalImage := func(imageName string) (bool, error) {
		return isImageLocal(runCtx, imageName)
	}
	labeller := label.NewLabeller(runCtx.AddSkaffoldLabels(), runCtx.CustomLabels())
	tester := getTester(runCtx, isLocalImage)
	syncer := getSyncer(runCtx)
	var deployer deploy.Deployer
	deployer, err = getDeployer(runCtx, labeller.Labels())
	if err != nil {
		return nil, fmt.Errorf("creating deployer: %w", err)
	}
	depLister := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
		buildDependencies, err := build.DependenciesForArtifact(ctx, artifact, runCtx, store)
		if err != nil {
			return nil, err
		}

		testDependencies, err := tester.TestDependencies()
		if err != nil {
			return nil, err
		}

		return append(buildDependencies, testDependencies...), nil
	}

	graph := build.ToArtifactGraph(runCtx.Artifacts())
	artifactCache, err := cache.NewCache(runCtx, isLocalImage, depLister, graph, store)
	if err != nil {
		return nil, fmt.Errorf("initializing cache: %w", err)
	}

	builder, tester, deployer = WithTimings(builder, tester, deployer, runCtx.CacheArtifacts())
	if runCtx.Notification() {
		deployer = WithNotification(deployer)
	}

	monitor := filemon.NewMonitor()
	intents, intentChan := setupIntents(runCtx)
	trigger, err := trigger.NewTrigger(runCtx, intents.IsAnyAutoEnabled)
	if err != nil {
		return nil, fmt.Errorf("creating watch trigger: %w", err)
	}

	return &SkaffoldRunner{
		builder:  builder,
		tester:   tester,
		deployer: deployer,
		tagger:   tagger,
		syncer:   syncer,
		monitor:  monitor,
		listener: &SkaffoldListener{
			Monitor:    monitor,
			Trigger:    trigger,
			intentChan: intentChan,
		},
		artifactStore: store,
		kubectlCLI:    kubectlCLI,
		labeller:      labeller,
		podSelector:   kubernetes.NewImageList(),
		cache:         artifactCache,
		runCtx:        runCtx,
		intents:       intents,
		isLocalImage:  isLocalImage,
	}, nil
}

func setupIntents(runCtx *runcontext.RunContext) (*intents, chan bool) {
	intents := newIntents(runCtx.AutoBuild(), runCtx.AutoSync(), runCtx.AutoDeploy())

	intentChan := make(chan bool, 1)
	setupTrigger("build", intents.setBuild, intents.setAutoBuild, intents.getAutoBuild, server.SetBuildCallback, server.SetAutoBuildCallback, intentChan)
	setupTrigger("sync", intents.setSync, intents.setAutoSync, intents.getAutoSync, server.SetSyncCallback, server.SetAutoSyncCallback, intentChan)
	setupTrigger("deploy", intents.setDeploy, intents.setAutoDeploy, intents.getAutoDeploy, server.SetDeployCallback, server.SetAutoDeployCallback, intentChan)

	return intents, intentChan
}

func setupTrigger(triggerName string, setIntent func(bool), setAutoTrigger func(bool), getAutoTrigger func() bool, singleTriggerCallback func(func()), autoTriggerCallback func(func(bool)), c chan<- bool) {
	setIntent(getAutoTrigger())
	// give the server a callback to set the intent value when a user request is received
	singleTriggerCallback(func() {
		if !getAutoTrigger() { //if auto trigger is disabled, we're in manual mode
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
	if pipeline.Build.LocalBuild.Push == nil {
		pushImages = cl.PushImages
		logrus.Debugf("push value not present, defaulting to %t because cluster.PushImages is %t", pushImages, cl.PushImages)
	} else {
		pushImages = *pipeline.Build.LocalBuild.Push
	}
	return !pushImages, nil
}

// getBuilder creates a builder from a given RunContext and build pipeline type.
func getBuilder(runCtx *runcontext.RunContext, store build.ArtifactStore, p latest.Pipeline) (build.PipelineBuilder, error) {
	switch {
	case p.Build.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		builder, err := local.NewBuilder(runCtx, p.Build.LocalBuild)
		if err != nil {
			return nil, err
		}
		builder.ArtifactStore(store)
		return builder, nil

	case p.Build.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		builder := gcb.NewBuilder(runCtx, p.Build.GoogleCloudBuild)
		builder.ArtifactStore(store)
		return builder, nil

	case p.Build.Cluster != nil:
		logrus.Debugln("Using builder: cluster")
		builder, err := cluster.NewBuilder(runCtx, p.Build.Cluster)
		if err != nil {
			return nil, err
		}
		builder.ArtifactStore(store)
		return builder, err

	default:
		return nil, fmt.Errorf("unknown builder for config %+v", p.Build)
	}
}

func getTester(cfg test.Config, isLocalImage func(imageName string) (bool, error)) test.Tester {
	return test.NewTester(cfg, isLocalImage)
}

func getSyncer(cfg sync.Config) sync.Syncer {
	return sync.NewSyncer(cfg)
}

func getDeployer(runCtx *runcontext.RunContext, labels map[string]string) (deploy.Deployer, error) {
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
	// avoid muxing overhead when only a single deployer is configured
	if len(deployers) == 1 {
		return deployers[0], nil
	}

	return deployers, nil
}
