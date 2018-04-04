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
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
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
	depMap     *build.DependencyMap
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

func newTaggerForConfig(t config.TagPolicy) (tag.Tagger, error) {
	if t.ShaTagger != nil {
		return &tag.ChecksumTagger{}, nil
	}
	if t.GitTagger != nil {
		return &tag.GitCommit{}, nil
	}

	return nil, fmt.Errorf("Unknown tagger for strategy %s", t)
}

// Run runs the skaffold build and deploy pipeline.
func (r *SkaffoldRunner) Run() error {
	ctx := context.Background()

	if r.opts.DevMode {
		return r.dev(ctx, r.config.Build.Artifacts)
	}

	_, _, err := r.buildAndDeploy(ctx, r.config.Build.Artifacts, nil)
	return err
}

func (r *SkaffoldRunner) dev(ctx context.Context, artifacts []*config.Artifact) error {
	var err error
	r.depMap, err = build.NewDependencyMap(artifacts, docker.DefaultDockerfileDepResolver)
	if err != nil {
		return errors.Wrap(err, "getting path to dependency map")
	}

	watcher, err := r.WatcherFactory(r.depMap.Paths())
	if err != nil {
		return err
	}

	podSelector := kubernetes.NewImageList()
	colorPicker := kubernetes.NewColorPicker(artifacts)
	logger := kubernetes.NewLogAggregator(r.opts.Output, podSelector, colorPicker)

	onBuildSuccess := func(bRes *build.BuildResult) {
		// Update which images are logged with which color
		for _, build := range bRes.Builds {
			podSelector.AddImage(build.Tag)
		}
	}

	onChange := func(changedPaths []string) {
		logger.Mute()

		_, _, err := r.buildAndDeploy(ctx, artifacts, onBuildSuccess)
		if err != nil {
			// In dev mode, we only warn on pipeline errors
			logrus.Warnf("run: %s", err)
		}

		fmt.Fprint(r.opts.Output, "Watching for changes...\n")
		logger.Unmute()
	}

	onChange(r.depMap.Paths())
	// Start logs
	if err = logger.Start(ctx, r.kubeclient.CoreV1()); err != nil {
		return err
	}

	// Watch files and rebuild
	watcher.Start(ctx, onChange)

	return nil
}

func (r *SkaffoldRunner) buildAndDeploy(ctx context.Context, artifacts []*config.Artifact, onBuildSuccess func(*build.BuildResult)) (*build.BuildResult, *deploy.Result, error) {
	bRes, err := r.build(ctx, artifacts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "build")
	}

	if onBuildSuccess != nil {
		onBuildSuccess(bRes)
	}

	// Make sure all artifacts are redeployed. Not only those that were just rebuilt.
	r.builds = mergeWithPreviousBuilds(bRes.Builds, r.builds)

	dRes, err := r.deploy(ctx, &build.BuildResult{
		Builds: r.builds,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "deploy")
	}

	return bRes, dRes, nil
}

func (r *SkaffoldRunner) build(ctx context.Context, artifacts []*config.Artifact) (*build.BuildResult, error) {
	start := time.Now()
	fmt.Fprintln(r.opts.Output, "Starting build...")

	bRes, err := r.Builder.Build(ctx, r.opts.Output, r.Tagger, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "build step")
	}

	fmt.Fprintln(r.opts.Output, "Build complete in", time.Now().Sub(start))

	return bRes, nil
}

func (r *SkaffoldRunner) deploy(ctx context.Context, bRes *build.BuildResult) (*deploy.Result, error) {
	start := time.Now()
	fmt.Fprintln(r.opts.Output, "Starting deploy...")

	dRes, err := r.Deployer.Deploy(ctx, r.opts.Output, bRes)
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
