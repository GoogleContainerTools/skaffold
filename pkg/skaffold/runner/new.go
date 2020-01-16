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

	"github.com/pkg/errors"
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
		return nil, errors.Wrap(err, "parsing tag config")
	}

	builder, err := getBuilder(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "parsing build config")
	}

	imagesAreLocal := false
	if localBuilder, ok := builder.(*local.Builder); ok {
		imagesAreLocal = !localBuilder.PushImages()
	}

	depLister := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
		return build.DependenciesForArtifact(ctx, artifact, runCtx.InsecureRegistries)
	}

	artifactCache, err := cache.NewCache(runCtx, imagesAreLocal, depLister)
	if err != nil {
		return nil, errors.Wrap(err, "initializing cache")
	}

	tester := getTester(runCtx)
	syncer := getSyncer(runCtx)

	deployer, err := getDeployer(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "parsing deploy config")
	}

	defaultLabeller := deploy.NewLabeller("")
	// runCtx.Opts is last to let users override/remove any label
	// deployer labels are added during deployment
	labellers := []deploy.Labeller{builder, tagger, defaultLabeller, &runCtx.Opts}

	builder, tester, deployer = WithTimings(builder, tester, deployer, runCtx.Opts.CacheArtifacts)
	if runCtx.Opts.Notification {
		deployer = WithNotification(deployer)
	}

	trigger, err := trigger.NewTrigger(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "creating watch trigger")
	}

	event.InitializeState(runCtx.Cfg.Build)

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
		return nil, errors.Wrapf(err, "setting up trigger callbacks")
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

func (r *SkaffoldRunner) setupTriggerCallback(triggerName string, c chan<- bool) error {
	var (
		setIntent      func(bool)
		trigger        bool
		serverCallback func(func())
	)

	switch triggerName {
	case "build":
		setIntent = r.intents.setBuild
		trigger = r.runCtx.Opts.AutoBuild
		serverCallback = server.SetBuildCallback
	case "sync":
		setIntent = r.intents.setSync
		trigger = r.runCtx.Opts.AutoSync
		serverCallback = server.SetSyncCallback
	case "deploy":
		setIntent = r.intents.setDeploy
		trigger = r.runCtx.Opts.AutoDeploy
		serverCallback = server.SetDeployCallback
	default:
		return fmt.Errorf("unsupported trigger type when setting callbacks: %s", triggerName)
	}

	setIntent(true)

	// if "auto" is set to false, we're in manual mode
	if !trigger {
		setIntent(false) // set the initial value of the intent to false
		// give the server a callback to set the intent value when a user request is received
		serverCallback(func() {
			logrus.Debugf("%s intent received, calling back to runner", triggerName)
			c <- true
			setIntent(true)
		})
	}
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

func getTester(runCtx *runcontext.RunContext) test.Tester {
	return test.NewTester(runCtx)
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
		return tag.NewGitCommit(t.GitTagger.Variant)

	case t.DateTimeTagger != nil:
		return tag.NewDateTimeTagger(t.DateTimeTagger.Format, t.DateTimeTagger.TimeZone), nil

	default:
		return nil, fmt.Errorf("unknown tagger for strategy %+v", t)
	}
}
