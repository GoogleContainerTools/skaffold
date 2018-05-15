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
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	clientgo "k8s.io/client-go/kubernetes"
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
	out        io.Writer
}

var kubernetesClient = kubernetes.GetClientset

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(opts *config.SkaffoldOptions, cfg *config.SkaffoldConfig, out io.Writer) (*SkaffoldRunner, error) {
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

	tagger, err := getTagger(cfg.Build.TagPolicy, opts.CustomTag)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold tag config")
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
		out:            out,
	}, nil
}

func getBuilder(cfg *v1alpha2.BuildConfig, kubeContext string) (build.Builder, error) {
	if cfg.LocalBuild != nil {
		logrus.Debugf("Using builder: local")
		return build.NewLocalBuilder(cfg, kubeContext)
	}
	if cfg.GoogleCloudBuild != nil {
		logrus.Debugf("Using builder: google cloud")
		return build.NewGoogleCloudBuilder(cfg)
	}
	if cfg.KanikoBuild != nil {
		logrus.Debugf("Using builder: kaniko")
		return build.NewKanikoBuilder(cfg)
	}

	return nil, fmt.Errorf("Unknown builder for config %+v", cfg)
}

func getDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) (deploy.Deployer, error) {
	if cfg.KubectlDeploy != nil {
		return deploy.NewKubectlDeployer(cfg, kubeContext), nil
	}
	if cfg.HelmDeploy != nil {
		return deploy.NewHelmDeployer(cfg, kubeContext), nil
	}

	return nil, fmt.Errorf("Unknown deployer for config %+v", cfg)
}

func getTagger(t v1alpha2.TagPolicy, customTag string) (tag.Tagger, error) {
	if customTag != "" {
		return &tag.CustomTag{
			Tag: customTag,
		}, nil
	}

	if t.EnvTemplateTagger != nil {
		return tag.NewEnvTemplateTagger(t.EnvTemplateTagger.Template)
	}
	if t.ShaTagger != nil {
		return &tag.ChecksumTagger{}, nil
	}
	if t.GitTagger != nil {
		return &tag.GitCommit{}, nil
	}

	return nil, fmt.Errorf("Unknown tagger for strategy %s", t)
}

// Build builds the artifacts.
func (r *SkaffoldRunner) Build(ctx context.Context) error {
	bRes, err := r.build(ctx, r.config.Build.Artifacts)
	if err != nil {
		return err
	}

	for _, res := range bRes.Builds {
		fmt.Fprintf(r.out, "%s -> %s\n", res.ImageName, res.Tag)
	}

	return nil
}

// Run runs the skaffold build and deploy pipeline.
func (r *SkaffoldRunner) Run(ctx context.Context) error {
	_, _, err := r.buildAndDeploy(ctx, r.config.Build.Artifacts, nil)
	return err
}

// Dev watches for changes and runs the skaffold build and deploy
// pipeline until interrrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context) error {
	if r.opts.Cleanup {
		return cleanUpOnCtrlC(ctx, r.watchBuildDeploy, r.cleanup)
	}
	return r.watchBuildDeploy(ctx)
}

func (r *SkaffoldRunner) watchBuildDeploy(ctx context.Context) error {
	artifacts := r.config.Build.Artifacts

	var err error
	r.depMap, err = build.NewDependencyMap(artifacts)
	if err != nil {
		return errors.Wrap(err, "getting path to dependency map")
	}

	watcher, err := r.WatcherFactory(r.depMap.Paths())
	if err != nil {
		return errors.Wrap(err, "creating watcher")
	}

	deployDeps, err := r.Deployer.Dependencies()
	if err != nil {
		return errors.Wrap(err, "getting deploy dependencies")
	}
	logrus.Infof("Deployer dependencies: %s", deployDeps)

	deployWatcher, err := r.WatcherFactory(deployDeps)
	if err != nil {
		return errors.Wrap(err, "creating deploy watcher")
	}

	podSelector := kubernetes.NewImageList()
	colorPicker := kubernetes.NewColorPicker(artifacts)
	logger := kubernetes.NewLogAggregator(r.out, podSelector, colorPicker)

	onBuildSuccess := func(bRes *build.BuildResult) {
		// Update which images are logged with which color
		for _, build := range bRes.Builds {
			podSelector.AddImage(build.Tag)
		}
	}

	onChange := func(changedPaths []string) {
		logger.Mute()

		changedArtifacts := r.depMap.ArtifactsForPaths(changedPaths)

		_, _, err := r.buildAndDeploy(ctx, changedArtifacts, onBuildSuccess)
		if err != nil {
			// In dev mode, we only log on pipeline errors
			logrus.Errorf("run: %s", err)
			logrus.Error("Skipping Deploy due to build error.")
		}

		fmt.Fprint(r.out, "Watching for changes...\n")
		logger.Unmute()
	}

	onDeployChange := func(changedPaths []string) {
		logger.Mute()
		_, err := r.deploy(ctx, &build.BuildResult{
			Builds: r.builds,
		})
		if err != nil {
			logrus.Warnf("deploy: %s", err)
		}
		fmt.Fprint(r.out, "Watching for changes...\n")
		logger.Unmute()
	}

	onChange(r.depMap.Paths())

	// Start logs
	if err = logger.Start(ctx, r.kubeclient.CoreV1()); err != nil {
		return errors.Wrap(err, "starting logger")
	}

	// Watch files and rebuild
	g, watchCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return watcher.Start(watchCtx, onChange)
	})
	g.Go(func() error {
		return deployWatcher.Start(watchCtx, onDeployChange)
	})

	return g.Wait()
}

func (r *SkaffoldRunner) buildAndDeploy(ctx context.Context, artifacts []*v1alpha2.Artifact, onBuildSuccess func(*build.BuildResult)) (*build.BuildResult, *deploy.Result, error) {
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

func (r *SkaffoldRunner) build(ctx context.Context, artifacts []*v1alpha2.Artifact) (*build.BuildResult, error) {
	start := time.Now()
	fmt.Fprintln(r.out, "Starting build...")

	bRes, err := r.Builder.Build(ctx, r.out, r.Tagger, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "build step")
	}

	fmt.Fprintln(r.out, "Build complete in", time.Since(start))

	return bRes, nil
}

func (r *SkaffoldRunner) deploy(ctx context.Context, bRes *build.BuildResult) (*deploy.Result, error) {
	start := time.Now()
	fmt.Fprintln(r.out, "Starting deploy...")

	dRes, err := r.Deployer.Deploy(ctx, r.out, bRes)
	if err != nil {
		return nil, errors.Wrap(err, "deploy step")
	}
	if r.opts.Notification {
		fmt.Fprint(r.out, constants.TerminalBell)
	}

	fmt.Fprintln(r.out, "Deploy complete in", time.Since(start))

	return dRes, nil
}

func cleanUpOnCtrlC(ctx context.Context, runDevMode func(context.Context) error, cleanup func(context.Context)) error {
	ctx, cancel := context.WithCancel(ctx)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGPIPE,
	)

	go func() {
		<-signals
		cancel()
	}()

	errRun := runDevMode(ctx)
	cleanup(ctx)
	return errRun
}

func (r *SkaffoldRunner) cleanup(ctx context.Context) {
	start := time.Now()
	fmt.Fprintln(r.out, "Cleaning up...")

	err := r.Deployer.Cleanup(ctx, r.out)
	if err != nil {
		logrus.Warnf("cleanup: %s", err)
		return
	}

	fmt.Fprintln(r.out, "Cleanup complete in", time.Since(start))
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
