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
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

// SkaffoldRunner is responsible for running the skaffold build and deploy pipeline.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	test.Tester
	tag.Tagger
	watch.Trigger
	sync.Syncer

	opts         *config.SkaffoldOptions
	watchFactory watch.Factory
	builds       []build.Artifact
	imageList    *kubernetes.ImageList
}

// NewForConfig returns a new SkaffoldRunner for a SkaffoldPipeline
func NewForConfig(opts *config.SkaffoldOptions, cfg *latest.SkaffoldPipeline) (*SkaffoldRunner, error) {
	kubeContext, err := kubectx.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current cluster context")
	}
	logrus.Infof("Using kubectl context: %s", kubeContext)

	defaultRepo, err := configutil.GetDefaultRepo(opts.DefaultRepo)
	if err != nil {
		return nil, errors.Wrap(err, "getting default repo")
	}

	tagger, err := getTagger(cfg.Build.TagPolicy, opts.CustomTag)
	if err != nil {
		return nil, errors.Wrap(err, "parsing tag config")
	}

	builder, err := getBuilder(&cfg.Build, kubeContext, opts)
	if err != nil {
		return nil, errors.Wrap(err, "parsing build config")
	}

	tester, err := getTester(&cfg.Test, opts)
	if err != nil {
		return nil, errors.Wrap(err, "parsing test config")
	}

	deployer, err := getDeployer(&cfg.Deploy, kubeContext, opts.Namespace, defaultRepo)
	if err != nil {
		return nil, errors.Wrap(err, "parsing deploy config")
	}

	deployer = deploy.WithLabels(deployer, opts, builder, deployer, tagger)
	builder, tester, deployer = WithTimings(builder, tester, deployer)
	if opts.Notification {
		deployer = WithNotification(deployer)
	}

	trigger, err := watch.NewTrigger(opts)
	if err != nil {
		return nil, errors.Wrap(err, "creating watch trigger")
	}

	return &SkaffoldRunner{
		Builder:      builder,
		Tester:       tester,
		Deployer:     deployer,
		Tagger:       tagger,
		Trigger:      trigger,
		Syncer:       &kubectl.Syncer{},
		opts:         opts,
		watchFactory: watch.NewWatcher,
		imageList:    kubernetes.NewImageList(),
	}, nil
}

func getBuilder(cfg *latest.BuildConfig, kubeContext string, opts *config.SkaffoldOptions) (build.Builder, error) {
	switch {
	case len(opts.PreBuiltImages) > 0:
		logrus.Debugln("Using pre-built images")
		return build.NewPreBuiltImagesBuilder(opts.PreBuiltImages), nil

	case cfg.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		return local.NewBuilder(cfg.LocalBuild, kubeContext)

	case cfg.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		return gcb.NewBuilder(cfg.GoogleCloudBuild), nil

	case cfg.KanikoBuild != nil:
		logrus.Debugln("Using builder: kaniko")
		return kaniko.NewBuilder(cfg.KanikoBuild)

	default:
		return nil, fmt.Errorf("unknown builder for config %+v", cfg)
	}
}

func getTester(cfg *latest.TestConfig, opts *config.SkaffoldOptions) (test.Tester, error) {
	switch {
	case len(opts.PreBuiltImages) > 0:
		logrus.Debugln("Skipping tests")
		return test.NewTester(&latest.TestConfig{})
	default:
		return test.NewTester(cfg)
	}
}

func getDeployer(cfg *latest.DeployConfig, kubeContext string, namespace string, defaultRepo string) (deploy.Deployer, error) {
	// TODO(dgageot): this should be the folder containing skaffold.yaml. Should also be moved elsewhere.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "finding current directory")
	}

	switch {
	case cfg.HelmDeploy != nil:
		return deploy.NewHelmDeployer(cfg.HelmDeploy, kubeContext, namespace, defaultRepo), nil

	case cfg.KubectlDeploy != nil:
		return deploy.NewKubectlDeployer(cwd, cfg.KubectlDeploy, kubeContext, namespace, defaultRepo), nil

	case cfg.KustomizeDeploy != nil:
		return deploy.NewKustomizeDeployer(cfg.KustomizeDeploy, kubeContext, namespace, defaultRepo), nil

	default:
		return nil, fmt.Errorf("unknown deployer for config %+v", cfg)
	}
}

func getTagger(t latest.TagPolicy, customTag string) (tag.Tagger, error) {
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
		return nil, fmt.Errorf("unknown tagger for strategy %+v", t)
	}
}

func (r *SkaffoldRunner) newLogger(out io.Writer, artifacts []*latest.Artifact) *kubernetes.LogAggregator {
	var imageNames []string
	for _, artifact := range artifacts {
		imageNames = append(imageNames, artifact.ImageName)
	}

	return kubernetes.NewLogAggregator(out, imageNames, r.imageList)
}

func (r *SkaffoldRunner) buildTestDeploy(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	bRes, err := r.BuildAndTest(ctx, out, artifacts)
	if err != nil {
		return err
	}

	// Update which images are logged.
	for _, build := range bRes {
		r.imageList.Add(build.Tag)
	}

	// Make sure all artifacts are redeployed. Not only those that were just built.
	r.builds = mergeWithPreviousBuilds(bRes, r.builds)

	if _, err := r.Deploy(ctx, out, r.builds); err != nil {
		return errors.Wrap(err, "deploy failed")
	}

	return nil
}

// Run builds artifacts, runs tests on built artifacts, and then deploys them.
func (r *SkaffoldRunner) Run(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) error {
	if err := r.buildTestDeploy(ctx, out, artifacts); err != nil {
		return err
	}

	if r.opts.Tail {
		logger := r.newLogger(out, artifacts)
		if err := logger.Start(ctx); err != nil {
			return errors.Wrap(err, "starting logger")
		}
		<-ctx.Done()
	}

	return nil
}

// BuildAndTest builds artifacts and runs tests on built artifacts
func (r *SkaffoldRunner) BuildAndTest(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	bRes, err := r.Build(ctx, out, r.Tagger, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "build failed")
	}

	if !r.opts.SkipTests {
		if err = r.Test(ctx, out, bRes); err != nil {
			return nil, errors.Wrap(err, "test failed")
		}
	}
	return bRes, err
}

// TailLogs prints the logs for deployed artifacts.
func (r *SkaffoldRunner) TailLogs(ctx context.Context, out io.Writer, artifacts []*latest.Artifact, bRes []build.Artifact) error {
	if !r.opts.Tail {
		return nil
	}

	for _, b := range bRes {
		r.imageList.Add(b.Tag)
	}

	logger := r.newLogger(out, artifacts)
	if err := logger.Start(ctx); err != nil {
		return errors.Wrap(err, "starting logger")
	}

	<-ctx.Done()
	return nil
}

// Dev watches for changes and runs the skaffold build and deploy
// pipeline until interrrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	logger := r.newLogger(out, artifacts)

	// Create watcher and register artifacts to build current state of files.
	changed := changes{}
	onChange := func() error {
		defer func() {
			changed.reset()
			r.Trigger.WatchForChanges(out)
		}()

		logger.Mute()

		for _, a := range changed.dirtyArtifacts {
			s, err := sync.NewItem(a.artifact, a.events, r.builds)
			if err != nil {
				return errors.Wrap(err, "sync")
			}
			if s != nil {
				changed.AddResync(s)
			} else {
				changed.AddRebuild(a.artifact)
			}
		}

		switch {
		case changed.needsReload:
			logger.Stop()
			return ErrorConfigurationChanged
		case len(changed.needsResync) > 0:
			for _, s := range changed.needsResync {
				color.Default.Fprintf(out, "Syncing %d files for %s\n", len(s.Copy)+len(s.Delete), s.Image)

				if err := r.Syncer.Sync(ctx, s); err != nil {
					logrus.Warnln("Skipping deploy due to sync error:", err)
					return nil
				}
			}
		case len(changed.needsRebuild) > 0:
			if err := r.buildTestDeploy(ctx, out, changed.needsRebuild); err != nil {
				logrus.Warnln("Skipping deploy due to errors:", err)
				return nil
			}
		case changed.needsRedeploy:
			if _, err := r.Deploy(ctx, out, r.builds); err != nil {
				logrus.Warnln("Skipping Deploy due to error:", err)
				return nil
			}
		}

		logger.Unmute()
		return nil
	}

	watcher := r.watchFactory()

	// Watch artifacts
	for i := range artifacts {
		artifact := artifacts[i]

		if !r.shouldWatch(artifact) {
			continue
		}

		if err := watcher.Register(
			func() ([]string, error) { return DependenciesForArtifact(ctx, artifact) },
			func(e watch.Events) { changed.AddDirtyArtifact(artifact, e) },
		); err != nil {
			return nil, errors.Wrapf(err, "watching files for artifact %s", artifact.ImageName)
		}
	}

	// Watch test configuration
	if err := watcher.Register(
		func() ([]string, error) { return r.TestDependencies() },
		func(watch.Events) { changed.needsRedeploy = true },
	); err != nil {
		return nil, errors.Wrap(err, "watching test files")
	}

	// Watch deployment configuration
	if err := watcher.Register(
		func() ([]string, error) { return r.Dependencies() },
		func(watch.Events) { changed.needsRedeploy = true },
	); err != nil {
		return nil, errors.Wrap(err, "watching files for deployer")
	}

	// Watch Skaffold configuration
	if err := watcher.Register(
		func() ([]string, error) { return []string{r.opts.ConfigurationFile}, nil },
		func(watch.Events) { changed.needsReload = true },
	); err != nil {
		return nil, errors.Wrapf(err, "watching skaffold configuration %s", r.opts.ConfigurationFile)
	}

	// First run
	if err := r.buildTestDeploy(ctx, out, artifacts); err != nil {
		return nil, errors.Wrap(err, "exiting dev mode because first run failed")
	}

	// Start logs
	if r.opts.TailDev {
		if err := logger.Start(ctx); err != nil {
			return nil, errors.Wrap(err, "starting logger")
		}
	}

	if r.opts.PortForward {
		portForwarder := kubernetes.NewPortForwarder(out, r.imageList)

		if err := portForwarder.Start(ctx); err != nil {
			return nil, errors.Wrap(err, "starting port-forwarder")
		}
	}

	r.Trigger.WatchForChanges(out)
	return nil, watcher.Run(ctx, r.Trigger, onChange)
}

func (r *SkaffoldRunner) shouldWatch(artifact *latest.Artifact) bool {
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
