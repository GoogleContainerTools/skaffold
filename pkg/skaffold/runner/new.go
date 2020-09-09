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
	kubectlCLI := kubectl.NewCLI(runCtx)

	tagger, err := getTagger(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating tagger: %w", err)
	}

	builder, imagesAreLocal, err := getBuilder(runCtx)
	if err != nil {
		return nil, fmt.Errorf("creating builder: %w", err)
	}

	labeller := deploy.NewLabeller(runCtx.AddSkaffoldLabels(), runCtx.CustomLabels())
	tester := getTester(runCtx, imagesAreLocal)
	syncer := getSyncer(runCtx)
	deployer := getDeployer(runCtx, labeller.Labels())

	depLister := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
		buildDependencies, err := build.DependenciesForArtifact(ctx, artifact, runCtx.GetInsecureRegistries())
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

	builder, tester, deployer = WithTimings(builder, tester, deployer, runCtx.CacheArtifacts())
	if runCtx.Notification() {
		deployer = WithNotification(deployer)
	}

	event.InitializeState(runCtx.Pipeline(), runCtx.GetKubeContext(), runCtx.AutoBuild(), runCtx.AutoDeploy(), runCtx.AutoSync())
	event.LogMetaEvent()

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

// getBuilder creates a builder from a given RunContext.
// Returns that builder, a bool to indicate that images are local
// (ie don't need to be pushed) and an error.
func getBuilder(runCtx *runcontext.RunContext) (build.Builder, bool, error) {
	b := runCtx.Pipeline().Build

	switch {
	case b.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		builder, err := local.NewBuilder(runCtx)
		if err != nil {
			return nil, false, err
		}
		return builder, !builder.PushImages(), nil

	case b.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		return gcb.NewBuilder(runCtx), false, nil

	case b.Cluster != nil:
		logrus.Debugln("Using builder: cluster")
		builder, err := cluster.NewBuilder(runCtx)
		return builder, false, err

	default:
		return nil, false, fmt.Errorf("unknown builder for config %+v", b)
	}
}

func getTester(cfg test.Config, imagesAreLocal bool) test.Tester {
	return test.NewTester(cfg, imagesAreLocal)
}

func getSyncer(cfg sync.Config) sync.Syncer {
	return sync.NewSyncer(cfg)
}

func getDeployer(cfg deploy.Config, labels map[string]string) deploy.Deployer {
	d := cfg.Pipeline().Deploy

	var deployers deploy.DeployerMux

	if d.HelmDeploy != nil {
		deployers = append(deployers, deploy.NewHelmDeployer(cfg, labels))
	}

	if d.KptDeploy != nil {
		deployers = append(deployers, deploy.NewKptDeployer(cfg, labels))
	}

	if d.KubectlDeploy != nil {
		deployers = append(deployers, deploy.NewKubectlDeployer(cfg, labels))
	}

	if d.KustomizeDeploy != nil {
		deployers = append(deployers, deploy.NewKustomizeDeployer(cfg, labels))
	}

	// avoid muxing overhead when only a single deployer is configured
	if len(deployers) == 1 {
		return deployers[0]
	}

	return deployers
}

func getTagger(runCtx *runcontext.RunContext) (tag.Tagger, error) {
	t := runCtx.Pipeline().Build.TagPolicy

	switch {
	case runCtx.CustomTag() != "":
		return &tag.CustomTag{
			Tag: runCtx.CustomTag(),
		}, nil

	case t.EnvTemplateTagger != nil:
		return tag.NewEnvTemplateTagger(t.EnvTemplateTagger.Template)

	case t.ShaTagger != nil:
		return &tag.ChecksumTagger{}, nil

	case t.GitTagger != nil:
		return tag.NewGitCommit(t.GitTagger.Prefix, t.GitTagger.Variant)

	case t.DateTimeTagger != nil:
		return tag.NewDateTimeTagger(t.DateTimeTagger.Format, t.DateTimeTagger.TimeZone), nil

	case t.CustomTemplateTagger != nil:
		components, err := CreateComponents(t.CustomTemplateTagger)

		if err != nil {
			return nil, fmt.Errorf("creating components: %w", err)
		}

		return tag.NewCustomTemplateTagger(t.CustomTemplateTagger.Template, components)

	default:
		return nil, fmt.Errorf("unknown tagger for strategy %+v", t)
	}
}

// CreateComponents creates a map of taggers for CustomTemplateTagger
func CreateComponents(t *latest.CustomTemplateTagger) (map[string]tag.Tagger, error) {
	components := map[string]tag.Tagger{}

	for _, taggerComponent := range t.Components {
		name, c := taggerComponent.Name, taggerComponent.Component

		if _, ok := components[name]; ok {
			return nil, fmt.Errorf("multiple components with name %s", name)
		}

		switch {
		case c.EnvTemplateTagger != nil:
			components[name], _ = tag.NewEnvTemplateTagger(c.EnvTemplateTagger.Template)

		case c.ShaTagger != nil:
			components[name] = &tag.ChecksumTagger{}

		case c.GitTagger != nil:
			components[name], _ = tag.NewGitCommit(c.GitTagger.Prefix, c.GitTagger.Variant)

		case c.DateTimeTagger != nil:
			components[name] = tag.NewDateTimeTagger(c.DateTimeTagger.Format, c.DateTimeTagger.TimeZone)

		case c.CustomTemplateTagger != nil:
			return nil, fmt.Errorf("nested customTemplate components are not supported in skaffold (%s)", name)

		default:
			return nil, fmt.Errorf("unknown component for custom template: %s %+v", name, c)
		}
	}

	return components, nil
}
