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
	tagger, err := getTagger(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating tagger: %w", err)
	}

	builder, err := getBuilder(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating builder: %w", err)
	}

	imagesAreLocal := false
	if localBuilder, ok := builder.(*local.Builder); ok {
		imagesAreLocal = !localBuilder.PushImages()
	}

	tester := getTester(runCtx, imagesAreLocal)
	syncer := getSyncer(runCtx)

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

	deployer, err := getDeployer(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating deployer: %w", err)
	}

	defaultLabeller := deploy.NewLabeller(runCtx.Opts)
	// runCtx.Opts is last to let users override/remove any label
	// deployer labels are added during deployment
	labellers := []deploy.Labeller{builder, tagger, defaultLabeller}

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

	intentChan := make(chan bool, 1)

	r := &SkaffoldRunner{
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
		changeSet: &changeSet{
			rebuildTracker: make(map[string]*latest.Artifact),
			resyncTracker:  make(map[string]*sync.Item),
		},
		labellers:            labellers,
		defaultLabeller:      defaultLabeller,
		portForwardResources: runCtx.Cfg.PortForward,
		podSelector:          kubernetes.NewImageList(),
		cache:                artifactCache,
		runCtx:               runCtx,
		intents:              newIntents(runCtx.Opts.AutoBuild, runCtx.Opts.AutoSync, runCtx.Opts.AutoDeploy),
		imagesAreLocal:       imagesAreLocal,
	}

	if err := r.setupTriggerCallbacks(intentChan); err != nil {
		return nil, fmt.Errorf("setting up trigger callbacks: %w", err)
	}

	return r, nil
}

func (r *SkaffoldRunner) setupTriggerCallbacks(c chan bool) error {
	if err := r.setupTriggerCallback("build", c); err != nil {
		return err
	}
	if err := r.setupTriggerCallback("sync", c); err != nil {
		return err
	}
	if err := r.setupTriggerCallback("deploy", c); err != nil {
		return err
	}

	return nil
}

func (r *SkaffoldRunner) setupUpdateAutoTriggerCallback(triggerName string, c chan<- bool) error {
	var (
		setIntent      func(bool)
		serverCallback func(func(bool))
	)

	switch triggerName {
	case "build":
		setIntent = r.intents.setAutoBuild
		serverCallback = server.SetAutoBuildCallback
	case "sync":
		setIntent = r.intents.setAutoSync
		serverCallback = server.SetAutoSyncCallback
	case "deploy":
		setIntent = r.intents.setAutoDeploy
		serverCallback = server.SetAutoDeployCallback
	default:
		return fmt.Errorf("unsupported trigger type when setting callbacks: %s", triggerName)
	}

	serverCallback(func(val bool) {
		logrus.Debugf("%s auto trigger update received, calling back to runner", triggerName)
		// signal chan only on resume requests
		if val {
			c <- true
		}
		setIntent(val)
	})
	return nil
}

func (r *SkaffoldRunner) setupTriggerCallback(triggerName string, c chan<- bool) error {
	var (
		setIntent             func(bool)
		setAutoTrigger        func(bool)
		trigger               func() bool
		singleTriggerCallback func(func())
		autoTriggerCallback   func(func(bool))
	)

	switch triggerName {
	case "build":
		setIntent = r.intents.setBuild
		setAutoTrigger = r.intents.setAutoBuild
		trigger = func() (b bool) {
			b, _, _ = r.intents.GetAutoTriggers()
			return
		}
		singleTriggerCallback = server.SetBuildCallback
		autoTriggerCallback = server.SetAutoBuildCallback
	case "sync":
		setIntent = r.intents.setSync
		setAutoTrigger = r.intents.setAutoSync
		trigger = func() (s bool) {
			_, s, _ = r.intents.GetAutoTriggers()
			return
		}
		singleTriggerCallback = server.SetSyncCallback
		autoTriggerCallback = server.SetAutoSyncCallback
	case "deploy":
		setIntent = r.intents.setDeploy
		setAutoTrigger = r.intents.setAutoDeploy
		trigger = func() (d bool) {
			_, _, d = r.intents.GetAutoTriggers()
			return
		}
		singleTriggerCallback = server.SetDeployCallback
		autoTriggerCallback = server.SetAutoDeployCallback
	default:
		return fmt.Errorf("unsupported trigger type when setting callbacks: %s", triggerName)
	}

	setIntent(trigger())

	// give the server a callback to set the intent value when a user request is received
	singleTriggerCallback(func() {
		if !trigger() { //if auto trigger is disabled, we're in manual mode
			logrus.Debugf("%s intent received, calling back to runner", triggerName)
			c <- true
			setIntent(true)
		}
	})

	// give the server a callback to set the intent and auto trigger values when a user request is received
	autoTriggerCallback(func(val bool) {
		logrus.Debugf("%s auto trigger update to %t received, calling back to runner", triggerName, val)
		// signal chan only on start auto trigger requests
		if val {
			c <- true
		}
		setAutoTrigger(val)
		setIntent(val)
	})

	return nil
}

func getBuilder(runCtx *runcontext.RunContext) (build.Builder, error) {
	switch {
	case runCtx.Cfg.Build.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		return local.NewBuilder(runCtx)

	case runCtx.Cfg.Build.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		return gcb.NewBuilder(runCtx), nil

	case runCtx.Cfg.Build.Cluster != nil:
		logrus.Debugln("Using builder: cluster")
		return cluster.NewBuilder(runCtx)

	default:
		return nil, fmt.Errorf("unknown builder for config %+v", runCtx.Cfg.Build)
	}
}

func getTester(runCtx *runcontext.RunContext, imagesAreLocal bool) test.Tester {
	return test.NewTester(runCtx, imagesAreLocal)
}

func getSyncer(runCtx *runcontext.RunContext) sync.Syncer {
	return sync.NewSyncer(runCtx)
}

func getDeployer(runCtx *runcontext.RunContext) (deploy.Deployer, error) {
	deployers := deploy.DeployerMux(nil)

	if runCtx.Cfg.Deploy.HelmDeploy != nil {
		deployers = append(deployers, deploy.NewHelmDeployer(runCtx))
	}

	if runCtx.Cfg.Deploy.KubectlDeploy != nil {
		deployers = append(deployers, deploy.NewKubectlDeployer(runCtx))
	}

	if runCtx.Cfg.Deploy.KustomizeDeploy != nil {
		deployers = append(deployers, deploy.NewKustomizeDeployer(runCtx))
	}

	if deployers == nil {
		return nil, fmt.Errorf("unknown deployer for config %+v", runCtx.Cfg.Deploy)
	}

	// avoid muxing overhead when only a single deployer is configured
	if len(deployers) == 1 {
		return deployers[0], nil
	}

	return deployers, nil
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
