/*
Copyright 2018 Google LLC

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
	"time"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/watch"
	clientgo "k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SkaffoldRunner is responsible for running the skaffold build and deploy pipeline.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	tag.Tagger
	watch.WatcherFactory

	opts       *config.SkaffoldOptions
	config     *config.SkaffoldConfig
	kubeclient clientgo.Interface
	builds     []build.Build
}

var kubernetesClient = kubernetes.GetClientset

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(opts *config.SkaffoldOptions, cfg *config.SkaffoldConfig) (*SkaffoldRunner, error) {
	kubeContext, err := kubernetes.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current cluster context")
	}
	logrus.Infof("Using kubectl context: %s", kubeContext)

	builder, err := getBuilder(&cfg.Build, kubeContext)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold build config")
	}
	deployer, err := getDeployer(&cfg.Deploy, kubeContext)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold deploy config")
	}
	tagger, err := newTaggerForConfig(cfg.Build.TagPolicy)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold tag config")
	}
	customTag := opts.CustomTag
	if customTag != "" {
		tagger = &tag.CustomTag{
			Tag: customTag,
		}
	}
	client, err := kubernetesClient()
	if err != nil {
		return nil, errors.Wrap(err, "getting k8s client")
	}
	return &SkaffoldRunner{
		config:         cfg,
		Builder:        builder,
		Deployer:       deployer,
		Tagger:         tagger,
		opts:           opts,
		kubeclient:     client,
		WatcherFactory: watch.NewWatcher,
	}, nil
}

func getBuilder(cfg *config.BuildConfig, kubeContext string) (build.Builder, error) {
	if cfg != nil && cfg.LocalBuild != nil {
		return build.NewLocalBuilder(cfg, kubeContext)
	}
	if cfg.GoogleCloudBuild != nil {
		return build.NewGoogleCloudBuilder(cfg)
	}

	return nil, fmt.Errorf("Unknown builder for config %+v", cfg)
}

func getDeployer(cfg *config.DeployConfig, kubeContext string) (deploy.Deployer, error) {
	if cfg.KubectlDeploy != nil {
		return deploy.NewKubectlDeployer(cfg, kubeContext), nil
	}
	if cfg.HelmDeploy != nil {
		return deploy.NewHelmDeployer(cfg, kubeContext), nil
	}

	return nil, fmt.Errorf("Unknown deployer for config %+v", cfg)
}

func newTaggerForConfig(tagStrategy string) (tag.Tagger, error) {
	switch tagStrategy {
	case constants.TagStrategySha256:
		return &tag.ChecksumTagger{}, nil
	case constants.TagStrategyGitCommit:
		return &tag.GitCommit{}, nil
	}

	return nil, fmt.Errorf("Unknown tagger for strategy %s", tagStrategy)
}

// Run runs the skaffold build and deploy pipeline.
func (r *SkaffoldRunner) Run() error {
	if r.opts.DevMode {
		return r.dev(r.config.Build.Artifacts)
	}

	if _, _, err := r.buildAndDeploy(r.config.Build.Artifacts); err != nil {
		return errors.Wrap(err, "run")
	}
	return nil
}

func (r *SkaffoldRunner) dev(artifacts []*config.Artifact) error {
	watcher, err := r.WatcherFactory(artifacts)
	if err != nil {
		return err
	}

	logger := kubernetes.NewLogAggregator(r.opts.Output)

	onChange := func(artifacts []*config.Artifact) {
		logger.SetCreationTime(time.Now())
		logger.Mute()

		bRes, _, err := r.buildAndDeploy(artifacts)
		if err != nil {
			// In dev mode, we only warn on pipeline errors
			logrus.Warnf("run: %s", err)
		}
		logger.Unmute()

		if bRes != nil {
			for i := range bRes.Builds {
				go logger.StreamLogs(r.kubeclient.CoreV1(), bRes.Builds[i].Tag)
			}
		}
		fmt.Fprint(r.opts.Output, "Watching for changes...\n")
	}

	onChange(artifacts)
	watcher.Start(context.Background(), onChange)

	return nil
}

func (r *SkaffoldRunner) buildAndDeploy(artifacts []*config.Artifact) (*build.BuildResult, *deploy.Result, error) {
	bRes, err := r.build(artifacts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "build step")
	}

	r.builds = mergeWithPreviousBuilds(bRes.Builds, r.builds)

	dRes, err := r.deploy(&build.BuildResult{
		Builds: r.builds,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "build step")
	}

	return bRes, dRes, nil
}

func (r *SkaffoldRunner) build(artifacts []*config.Artifact) (*build.BuildResult, error) {
	start := time.Now()
	fmt.Fprintln(r.opts.Output, "Starting build...")

	bRes, err := r.Builder.Build(r.opts.Output, r.Tagger, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "build step")
	}

	fmt.Fprintln(r.opts.Output, "Build complete in", time.Now().Sub(start))

	return bRes, nil
}

func (r *SkaffoldRunner) deploy(bRes *build.BuildResult) (*deploy.Result, error) {
	start := time.Now()
	fmt.Fprintln(r.opts.Output, "Starting deploy...")

	dRes, err := r.Deployer.Deploy(r.opts.Output, bRes)
	if err != nil {
		return nil, errors.Wrap(err, "deploy step")
	}
	if r.opts.Notification {
		fmt.Fprint(r.opts.Output, constants.TerminalBell)
	}

	fmt.Fprintln(r.opts.Output, "Deploy complete in", time.Now().Sub(start))

	return dRes, nil
}

func mergeWithPreviousBuilds(builds, previous []build.Build) []build.Build {
	updatedBuilds := map[string]bool{}
	for _, build := range builds {
		updatedBuilds[build.ImageName] = true
	}

	var merged []build.Build
	merged = append(merged, builds...)

	for _, b := range previous {
		if !updatedBuilds[b.ImageName] {
			merged = append(merged, b)
		}
	}

	return merged
}
