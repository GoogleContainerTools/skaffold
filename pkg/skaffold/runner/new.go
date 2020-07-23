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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
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
	kubectlCLI := kubectl.NewFromRunContext(runCtx)

	tagger, err := getTagger(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating tagger: %w", err)
	}

	builder, imagesAreLocal, err := getBuilder(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating builder: %w", err)
	}

	labeller := deploy.NewLabeller(runCtx.Opts.AddSkaffoldLabels, runCtx.Opts.CustomLabels)
	tester := getTester(runCtx, imagesAreLocal)
	syncer := getSyncer(runCtx)
	deployer := getDeployer(runCtx, labeller.Labels())

	depLister := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
		buildDependencies, err := build.DependenciesForArtifact(ctx, artifact, runCtx.InsecureRegistries)
		if err != nil {
			return nil, err
		}

		testDependencies, err := tester.TestDependencies()
		if err != nil {
			return nil, err
		}

		return append(buildDependencies, testDependencies...), nil
	}

	artifactCache, err := cache.NewCache(runCtx, imagesAreLocal, depLister)
	if err != nil {
		return nil, fmt.Errorf("initializing cache: %w", err)
	}

	builder, tester, deployer = WithTimings(builder, tester, deployer, runCtx.Opts.CacheArtifacts)
	if runCtx.Opts.Notification {
		deployer = WithNotification(deployer)
	}

	trigger, err := trigger.NewTrigger(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating watch trigger: %w", err)
	}

	event.InitializeState(runCtx.Cfg, runCtx.KubeContext, runCtx.Opts.AutoBuild, runCtx.Opts.AutoDeploy, runCtx.Opts.AutoSync)
	event.LogMetaEvent()

	monitor := filemon.NewMonitor()
	intents, intentChan := setupIntents(runCtx)

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
		kubectlCLI:     kubectlCLI,
		labeller:       labeller,
		podSelector:    kubernetes.NewImageList(),
		cache:          artifactCache,
		runCtx:         runCtx,
		intents:        intents,
		imagesAreLocal: imagesAreLocal,
	}, nil
}

func setupIntents(runCtx *runcontext.RunContext) (*intents, chan bool) {
	intents := newIntents(runCtx.Opts.AutoBuild, runCtx.Opts.AutoSync, runCtx.Opts.AutoDeploy)

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

// getBuilder creates a builder from a given RunContext.
// Returns that builder, a bool to indicate that images are local
// (ie don't need to be pushed) and an error.
func getBuilder(runCtx *runcontext.RunContext) (build.Builder, bool, error) {
	switch {
	case runCtx.Cfg.Build.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		builder, err := local.NewBuilder(runCtx)
		if err != nil {
			return nil, false, err
		}
		return builder, !builder.PushImages(), nil

	case runCtx.Cfg.Build.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		return gcb.NewBuilder(runCtx), false, nil

	case runCtx.Cfg.Build.Cluster != nil:
		logrus.Debugln("Using builder: cluster")
		builder, err := cluster.NewBuilder(runCtx)
		return builder, false, err

	default:
		return nil, false, fmt.Errorf("unknown builder for config %+v", runCtx.Cfg.Build)
	}
}

func getTester(runCtx *runcontext.RunContext, imagesAreLocal bool) test.Tester {
	return test.NewTester(runCtx, imagesAreLocal)
}

func getSyncer(runCtx *runcontext.RunContext) sync.Syncer {
	return sync.NewSyncer(runCtx)
}

func getDeployer(runCtx *runcontext.RunContext, labels map[string]string) deploy.Deployer {
	var deployers deploy.DeployerMux

	if runCtx.Cfg.Deploy.HelmDeploy != nil {
		deployers = append(deployers, deploy.NewHelmDeployer(runCtx, labels))
	}

	if runCtx.Cfg.Deploy.KubectlDeploy != nil {
		deployers = append(deployers, deploy.NewKubectlDeployer(runCtx, labels))
	}

	if runCtx.Cfg.Deploy.KustomizeDeploy != nil {
		deployers = append(deployers, deploy.NewKustomizeDeployer(runCtx, labels))
	}

	// avoid muxing overhead when only a single deployer is configured
	if len(deployers) == 1 {
		return deployers[0]
	}

	return deployers
}

func getTagger(runCtx *runcontext.RunContext) (tag.Tagger, error) {
	t := runCtx.Cfg.Build.TagPolicy

	switch {
	case runCtx.Opts.CustomTag != "":
		return &tag.CustomTag{
			Tag: runCtx.Opts.CustomTag,
		}, nil

	case t.EnvTemplateTagger != nil:
		return tag.NewEnvTemplateTagger(t.EnvTemplateTagger.Template)

	case t.ShaTagger != nil:
		return &tag.ChecksumTagger{}, nil

	case t.GitTagger != nil:
		return tag.NewGitCommit(t.GitTagger.Prefix, t.GitTagger.Variant)

	case t.DateTimeTagger != nil:
		return tag.NewDateTimeTagger(t.DateTimeTagger.Format, t.DateTimeTagger.TimeZone), nil

	default:
		return nil, fmt.Errorf("unknown tagger for strategy %+v", t)
	}
}
