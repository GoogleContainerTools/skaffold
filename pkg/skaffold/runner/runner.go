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
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/environments/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
)

// SkaffoldRunner is responsible for running the skaffold build and deploy pipeline.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	test.Tester
	tag.Tagger
	sync.Syncer
	watch.Watcher

	opts              *config.SkaffoldOptions
	labellers         []deploy.Labeller
	builds            []build.Artifact
	hasDeployed       bool
	needsPush         bool
	imageList         *kubernetes.ImageList
	namespaces        []string
	RPCServerShutdown func() error
}

// NewForConfig returns a new SkaffoldRunner for a SkaffoldPipeline
func NewForConfig(opts *config.SkaffoldOptions, cfg *latest.SkaffoldPipeline, kubeContext string) (*SkaffoldRunner, error) {
	logrus.Infof("Using kubectl context: %s", kubeContext)

	namespaces, err := getAllPodNamespaces(opts.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "getting namespace list")
	}

	defaultRepo, err := configutil.GetDefaultRepo(opts.DefaultRepo, kubeContext)
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

	tester, err := getTester(cfg.Test, opts)
	if err != nil {
		return nil, errors.Wrap(err, "parsing test config")
	}

	deployer, err := getDeployer(&cfg.Deploy, kubeContext, opts.Namespace, defaultRepo)
	if err != nil {
		return nil, errors.Wrap(err, "parsing deploy config")
	}

	labellers := []deploy.Labeller{opts, builder, deployer, tagger}

	builder, tester, deployer = WithTimings(builder, tester, deployer, opts.CacheArtifacts)
	if opts.Notification {
		deployer = WithNotification(deployer)
	}

	trigger, err := watch.NewTrigger(opts)
	if err != nil {
		return nil, errors.Wrap(err, "creating watch trigger")
	}

	shutdown, err := event.InitializeState(&cfg.Build, &cfg.Deploy, opts)
	if err != nil {
		return nil, errors.Wrap(err, "initializing skaffold event handler")
	}

	event.LogSkaffoldMetadata(version.Get())

	return &SkaffoldRunner{
		Builder:           builder,
		Tester:            tester,
		Deployer:          deployer,
		Tagger:            tagger,
		Syncer:            kubectl.NewSyncer(namespaces),
		Watcher:           watch.NewWatcher(trigger),
		opts:              opts,
		labellers:         labellers,
		imageList:         kubernetes.NewImageList(),
		namespaces:        namespaces,
		needsPush:         needsPush(cfg.Build),
		RPCServerShutdown: shutdown,
	}, nil
}

func getBuilder(cfg *latest.BuildConfig, kubeContext string, opts *config.SkaffoldOptions) (build.Builder, error) {
	switch {
	case buildWithPlugin(cfg.Artifacts):
		logrus.Debugln("Using builder plugins")
		return plugin.NewPluginBuilder(cfg, opts)
	case len(opts.PreBuiltImages) > 0:
		logrus.Debugln("Using pre-built images")
		return build.NewPreBuiltImagesBuilder(opts.PreBuiltImages), nil

	case cfg.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		return local.NewBuilder(cfg.LocalBuild, kubeContext, opts.SkipTests)

	case cfg.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		return gcb.NewBuilder(cfg.GoogleCloudBuild, opts.SkipTests), nil

	case cfg.KanikoBuild != nil:
		logrus.Debugln("Using builder: kaniko")
		return kaniko.NewBuilder(cfg.KanikoBuild)

	default:
		return nil, fmt.Errorf("unknown builder for config %+v", cfg)
	}
}

func needsPush(cfg latest.BuildConfig) bool {
	if cfg.LocalBuild == nil {
		return false
	}
	if cfg.LocalBuild.Push == nil {
		return false
	}
	return *cfg.LocalBuild.Push
}

func buildWithPlugin(artifacts []*latest.Artifact) bool {
	for _, a := range artifacts {
		if a.BuilderPlugin != nil {
			return true
		}
	}
	return false
}

func getTester(cfg []*latest.TestCase, opts *config.SkaffoldOptions) (test.Tester, error) {
	switch {
	case len(opts.PreBuiltImages) > 0:
		logrus.Debugln("Skipping tests")
		return test.NewTester(nil)
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

	return kubernetes.NewLogAggregator(out, imageNames, r.imageList, r.namespaces)
}

// HasDeployed returns true if this runner has deployed something.
func (r *SkaffoldRunner) HasDeployed() bool {
	return r.hasDeployed
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

	if err := r.Deploy(ctx, out, r.builds); err != nil {
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

// imageTags generates tags for a list of artifacts
func (r *SkaffoldRunner) imageTags(out io.Writer, artifacts []*latest.Artifact) (tag.ImageTags, error) {
	start := time.Now()
	color.Default.Fprintln(out, "Generating tags...")

	tags := make(tag.ImageTags, len(artifacts))

	for _, artifact := range artifacts {
		imageName := artifact.ImageName
		color.Default.Fprintf(out, " - %s -> ", imageName)

		tag, err := r.Tagger.GenerateFullyQualifiedImageName(artifact.Workspace, imageName)
		if err != nil {
			return nil, errors.Wrapf(err, "generating tag for %s", imageName)
		}

		fmt.Fprintln(out, tag)

		tags[imageName] = tag
	}

	color.Default.Fprintln(out, "Tags generated in", time.Since(start))
	return tags, nil
}

// BuildAndTest builds artifacts and runs tests on built artifacts
func (r *SkaffoldRunner) BuildAndTest(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	tags, err := r.imageTags(out, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "generating tag")
	}

	artifactCache := build.NewCache(ctx, r.Builder, r.opts, r.needsPush)
	artifactsToBuild, res := artifactCache.RetrieveCachedArtifacts(ctx, out, artifacts)
	bRes, err := r.Build(ctx, out, tags, artifactsToBuild)
	if err != nil {
		return nil, errors.Wrap(err, "build failed")
	}
	artifactCache.Retag(ctx, out, artifactsToBuild, bRes)
	bRes = append(bRes, res...)
	if err := artifactCache.CacheArtifacts(ctx, artifacts, bRes); err != nil {
		logrus.Warnf("error caching artifacts: %v", err)
	}
	if !r.opts.SkipTests {
		if err = r.Test(ctx, out, bRes); err != nil {
			return nil, errors.Wrap(err, "test failed")
		}
	}
	return bRes, err
}

// Deploy deploys the given artifacts
func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	err := r.Deployer.Deploy(ctx, out, artifacts, r.labellers)
	r.hasDeployed = true
	return err
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

func getAllPodNamespaces(configNamespace string) ([]string, error) {
	// We also get the default namespace.
	nsMap := make(map[string]bool)
	if configNamespace == "" {
		config, err := kubectx.CurrentConfig()
		if err != nil {
			return nil, errors.Wrap(err, "getting k8s configuration")
		}
		context, ok := config.Contexts[config.CurrentContext]
		if ok {
			nsMap[context.Namespace] = true
		} else {
			nsMap[""] = true
		}
	} else {
		nsMap[configNamespace] = true
	}

	// FIXME: Set additional namespaces from the selected yamls.

	// Collate the slice of namespaces.
	namespaces := make([]string, 0, len(nsMap))
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}
