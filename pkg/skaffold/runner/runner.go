/*
Copyright 2018 The Skaffold Authors

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
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const PollInterval = 1000 * time.Millisecond

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

// SkaffoldRunner is responsible for running the skaffold build and deploy pipeline.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	tag.Tagger

	opts         *config.SkaffoldOptions
	watchFactory watch.Factory
	builds       []build.Artifact
}

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(opts *config.SkaffoldOptions, cfg *config.SkaffoldConfig) (*SkaffoldRunner, error) {
	kubeContext, err := kubectx.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current cluster context")
	}
	logrus.Infof("Using kubectl context: %s", kubeContext)

	tagger, err := getTagger(cfg.Build.TagPolicy, opts.CustomTag)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold tag config")
	}

	builder, err := getBuilder(&cfg.Build, kubeContext)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold build config")
	}

	deployer, err := getDeployer(&cfg.Deploy, kubeContext, opts.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold deploy config")
	}

	deployer = deploy.WithLabels(deployer, opts, builder, deployer, tagger)
	builder, deployer = WithTimings(builder, deployer)
	if opts.Notification {
		deployer = WithNotification(deployer)
	}

	return &SkaffoldRunner{
		Builder:      builder,
		Deployer:     deployer,
		Tagger:       tagger,
		opts:         opts,
		watchFactory: watch.NewWatcher,
	}, nil
}

func getBuilder(cfg *v1alpha2.BuildConfig, kubeContext string) (build.Builder, error) {
	switch {
	case cfg.LocalBuild != nil:
		logrus.Debugf("Using builder: local")
		return local.NewBuilder(cfg.LocalBuild, kubeContext)

	case cfg.GoogleCloudBuild != nil:
		logrus.Debugf("Using builder: google cloud")
		return gcb.NewBuilder(cfg.GoogleCloudBuild), nil

	case cfg.KanikoBuild != nil:
		logrus.Debugf("Using builder: kaniko")
		return kaniko.NewBuilder(cfg.KanikoBuild), nil

	default:
		return nil, fmt.Errorf("Unknown builder for config %+v", cfg)
	}
}

func getDeployer(cfg *v1alpha2.DeployConfig, kubeContext string, namespace string) (deploy.Deployer, error) {
	switch {
	case cfg.KubectlDeploy != nil:
		// TODO(dgageot): this should be the folder containing skaffold.yaml. Should also be moved elsewhere.
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "finding current directory")
		}
		return deploy.NewKubectlDeployer(cwd, cfg.KubectlDeploy, kubeContext, namespace), nil

	case cfg.HelmDeploy != nil:
		return deploy.NewHelmDeployer(cfg.HelmDeploy, kubeContext, namespace), nil

	case cfg.KustomizeDeploy != nil:
		return deploy.NewKustomizeDeployer(cfg.KustomizeDeploy, kubeContext, namespace), nil

	default:
		return nil, fmt.Errorf("Unknown deployer for config %+v", cfg)
	}
}

func getTagger(t v1alpha2.TagPolicy, customTag string) (tag.Tagger, error) {
	switch {
	case customTag != "":
		return &tag.CustomTag{
			Tag: customTag,
		}, nil

	case t.EnvTemplateTagger != nil:
		return tag.NewEnvTemplateTagger(t.EnvTemplateTagger.Template)

	case t.ShaTagger != nil:
		return &tag.ChecksumTagger{}, nil

	case t.GitTagger != nil:
		return &tag.GitCommit{}, nil

	case t.DateTimeTagger != nil:
		return tag.NewDateTimeTagger(t.DateTimeTagger.Format, t.DateTimeTagger.TimeZone), nil

	default:
		return nil, fmt.Errorf("Unknown tagger for strategy %+v", t)
	}
}

// Run builds artifacts and then deploys them.
func (r *SkaffoldRunner) Run(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact) error {
	bRes, err := r.Build(ctx, out, r.Tagger, artifacts)
	if err != nil {
		return errors.Wrap(err, "build step")
	}

	_, err = r.Deploy(ctx, out, bRes)
	if err != nil {
		return errors.Wrap(err, "deploy step")
	}

	return r.TailLogs(ctx, out, artifacts, bRes)
}

// TailLogs prints the logs for deployed artifacts.
func (r *SkaffoldRunner) TailLogs(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact, bRes []build.Artifact) error {
	if !r.opts.Tail {
		return nil
	}

	imageList := kubernetes.NewImageList()
	for _, b := range bRes {
		imageList.Add(b.Tag)
	}

	colorPicker := kubernetes.NewColorPicker(artifacts)
	logger := kubernetes.NewLogAggregator(out, imageList, colorPicker)
	if err := logger.Start(ctx); err != nil {
		return errors.Wrap(err, "starting logger")
	}

	<-ctx.Done()
	return nil
}

// Dev watches for changes and runs the skaffold build and deploy
// pipeline until interrrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact) ([]build.Artifact, error) {
	imageList := kubernetes.NewImageList()
	colorPicker := kubernetes.NewColorPicker(artifacts)
	logger := kubernetes.NewLogAggregator(out, imageList, colorPicker)

	// Create watcher and register artifacts to build current state of files.
	changed := changes{}
	onChange := func() error {
		logger.Mute()

		var err error

		switch {
		case changed.needsReload:
			err = ErrorConfigurationChanged
		case len(changed.dirtyArtifacts) > 0:
			err = r.buildAndDeploy(ctx, out, changed.dirtyArtifacts, imageList)
		case changed.needsRedeploy:
			if _, err := r.Deploy(ctx, out, r.builds); err != nil {
				logrus.Warnln("Skipping Deploy due to error:", err)
			}
		}

		color.Default.Fprintln(out, "Watching for changes...")
		changed.reset()
		logger.Unmute()

		return err
	}

	watcher := r.watchFactory()

	// Watch artifacts
	for i := range artifacts {
		artifact := artifacts[i]

		if !r.shouldWatch(artifact) {
			continue
		}

		if err := watcher.Register(
			func() ([]string, error) { return dependenciesForArtifact(artifact) },
			func() { changed.Add(artifact) },
		); err != nil {
			return nil, errors.Wrapf(err, "watching files for artifact %s", artifact.ImageName)
		}
	}

	// Watch deployment configuration
	if err := watcher.Register(
		func() ([]string, error) { return r.Dependencies() },
		func() { changed.needsRedeploy = true },
	); err != nil {
		return nil, errors.Wrap(err, "watching files for deployer")
	}

	// Watch Skaffold configuration
	if err := watcher.Register(
		func() ([]string, error) { return []string{r.opts.ConfigurationFile}, nil },
		func() { changed.needsReload = true },
	); err != nil {
		return nil, errors.Wrapf(err, "watching skaffold configuration %s", r.opts.ConfigurationFile)
	}

	// First run
	if err := r.buildAndDeploy(ctx, out, artifacts, imageList); err != nil {
		return nil, errors.Wrap(err, "first run")
	}

	// Start logs
	if err := logger.Start(ctx); err != nil {
		return nil, errors.Wrap(err, "starting logger")
	}

	return nil, watcher.Run(ctx, PollInterval, onChange)
}

func (r *SkaffoldRunner) shouldWatch(artifact *v1alpha2.Artifact) bool {
	if len(r.opts.Watch) == 0 {
		return true
	}

	for _, watchExpression := range r.opts.Watch {
		if strings.Contains(artifact.ImageName, watchExpression) {
			return true
		}
	}

	return false
}

// buildAndDeploy builds a subset of the artifacts and deploys everything.
func (r *SkaffoldRunner) buildAndDeploy(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact, images *kubernetes.ImageList) error {
	firstRun := r.builds == nil

	bRes, err := r.Build(ctx, out, r.Tagger, artifacts)
	if err != nil {
		if firstRun {
			return errors.Wrap(err, "exiting dev mode because the first build failed")
		}

		logrus.Warnln("Skipping Deploy due to build error:", err)
		return nil
	}

	// Update which images are logged.
	for _, build := range bRes {
		images.Add(build.Tag)
	}

	// Make sure all artifacts are redeployed. Not only those that were just rebuilt.
	r.builds = mergeWithPreviousBuilds(bRes, r.builds)

	_, err = r.Deploy(ctx, out, r.builds)
	if err != nil {
		if firstRun {
			return errors.Wrap(err, "exiting dev mode because the first deploy failed")
		}

		logrus.Warnln("Skipping Deploy due to error:", err)
		return nil
	}

	return nil
}

func mergeWithPreviousBuilds(builds, previous []build.Artifact) []build.Artifact {
	updatedBuilds := map[string]bool{}
	for _, build := range builds {
		updatedBuilds[build.ImageName] = true
	}

	var merged []build.Artifact
	merged = append(merged, builds...)

	for _, b := range previous {
		if !updatedBuilds[b.ImageName] {
			merged = append(merged, b)
		}
	}

	return merged
}

func dependenciesForArtifact(a *v1alpha2.Artifact) ([]string, error) {
	var (
		paths []string
		err   error
	)

	switch {
	case a.DockerArtifact != nil:
		paths, err = docker.GetDependencies(a.Workspace, a.DockerArtifact)

	case a.BazelArtifact != nil:
		paths, err = bazel.GetDependencies(a.Workspace, a.BazelArtifact)

	default:
		return nil, fmt.Errorf("undefined artifact type: %+v", a.ArtifactType)
	}

	if err != nil {
		return nil, err
	}

	var p []string
	for _, path := range paths {
		p = append(p, filepath.Join(a.Workspace, path))
	}
	return p, nil
}
