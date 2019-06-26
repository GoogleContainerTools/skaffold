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
	"time"

	cfg "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Runner is responsible for running the skaffold build, test and deploy config.
type Runner interface {
	DiagnoseArtifacts(io.Writer) error
	Dev(context.Context, io.Writer, []*latest.Artifact) error
	BuildAndTest(context.Context, io.Writer, []*latest.Artifact) ([]build.Artifact, error)
	DeployAndLog(context.Context, io.Writer, []build.Artifact) error
	Cleanup(context.Context, io.Writer) error
	Prune(context.Context, io.Writer) error
	HasDeployed() bool
	HasBuilt() bool
	Stop() error
}

// SkaffoldRunner is responsible for running the skaffold build, test and deploy config.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	test.Tester
	tag.Tagger
	sync.Syncer
	watch.Watcher

	cache             *cache.Cache
	runCtx            *runcontext.RunContext
	labellers         []deploy.Labeller
	defaultLabeller   *deploy.DefaultLabeller
	builds            []build.Artifact
	hasBuilt          bool
	hasDeployed       bool
	imageList         *kubernetes.ImageList
	RPCServerShutdown func() error
}

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(opts *config.SkaffoldOptions, cfg *latest.SkaffoldConfig) (*SkaffoldRunner, error) {
	runCtx, err := runcontext.GetRunContext(opts, &cfg.Pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "getting run context")
	}

	tagger, err := getTagger(cfg.Build.TagPolicy, opts.CustomTag)
	if err != nil {
		return nil, errors.Wrap(err, "parsing tag config")
	}

	builder, err := getBuilder(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "parsing build config")
	}
	artifactCache := cache.NewCache(builder, runCtx)

	tester := getTester(runCtx)

	deployer, err := getDeployer(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "parsing deploy config")
	}

	defaultLabeller := deploy.NewLabeller("")
	labellers := []deploy.Labeller{opts, builder, deployer, tagger, defaultLabeller}

	builder, tester, deployer = WithTimings(builder, tester, deployer, opts.CacheArtifacts)
	if opts.Notification {
		deployer = WithNotification(deployer)
	}

	trigger, err := watch.NewTrigger(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "creating watch trigger")
	}

	shutdown, err := server.Initialize(opts)
	if err != nil {
		return nil, errors.Wrap(err, "initializing skaffold server")
	}
	event.InitializeState(runCtx)
	event.LogSkaffoldMetadata(version.Get())

	return &SkaffoldRunner{
		Builder:           builder,
		Tester:            tester,
		Deployer:          deployer,
		Tagger:            tagger,
		Syncer:            kubectl.NewSyncer(runCtx.Namespaces),
		Watcher:           watch.NewWatcher(trigger),
		labellers:         labellers,
		defaultLabeller:   defaultLabeller,
		imageList:         kubernetes.NewImageList(),
		cache:             artifactCache,
		runCtx:            runCtx,
		RPCServerShutdown: shutdown,
	}, nil
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

func getDeployer(runCtx *runcontext.RunContext) (deploy.Deployer, error) {
	switch {
	case runCtx.Cfg.Deploy.HelmDeploy != nil:
		return deploy.NewHelmDeployer(runCtx), nil

	case runCtx.Cfg.Deploy.KubectlDeploy != nil:
		return deploy.NewKubectlDeployer(runCtx), nil

	case runCtx.Cfg.Deploy.KustomizeDeploy != nil:
		return deploy.NewKustomizeDeployer(runCtx), nil

	default:
		return nil, fmt.Errorf("unknown deployer for config %+v", runCtx.Cfg.Deploy)
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
		return tag.NewGitCommit(t.GitTagger.Variant)

	case t.DateTimeTagger != nil:
		return tag.NewDateTimeTagger(t.DateTimeTagger.Format, t.DateTimeTagger.TimeZone), nil

	default:
		return nil, fmt.Errorf("unknown tagger for strategy %+v", t)
	}
}

func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	if cfg.IsKindCluster(r.runCtx.KubeContext) {
		// With `kind`, docker images have to be loaded with the `kind` CLI.
		if err := r.loadImagesInKindNodes(ctx, out, artifacts); err != nil {
			return errors.Wrapf(err, "loading images into kind nodes")
		}
	}

	err := r.Deployer.Deploy(ctx, out, artifacts, r.labellers)
	r.hasDeployed = true
	return err
}

// HasDeployed returns true if this runner has deployed something.
func (r *SkaffoldRunner) HasDeployed() bool {
	return r.hasDeployed
}

// HasBuilt returns true if this runner has built something.
func (r *SkaffoldRunner) HasBuilt() bool {
	return r.hasBuilt
}

type tagErr struct {
	tag string
	err error
}

// imageTags generates tags for a list of artifacts
func (r *SkaffoldRunner) imageTags(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) (tag.ImageTags, error) {
	start := time.Now()
	color.Default.Fprintln(out, "Generating tags...")

	tagErrs := make([]chan tagErr, len(artifacts))

	for i := range artifacts {
		tagErrs[i] = make(chan tagErr, 1)

		i := i
		go func() {
			tag, err := r.Tagger.GenerateFullyQualifiedImageName(artifacts[i].Workspace, artifacts[i].ImageName)
			tagErrs[i] <- tagErr{tag: tag, err: err}
		}()
	}

	imageTags := make(tag.ImageTags, len(artifacts))

	for i, artifact := range artifacts {
		imageName := artifact.ImageName
		color.Default.Fprintf(out, " - %s -> ", imageName)

		select {
		case <-ctx.Done():
			return nil, context.Canceled

		case t := <-tagErrs[i]:
			tag := t.tag
			err := t.err
			if err != nil {
				return nil, errors.Wrapf(err, "generating tag for %s", imageName)
			}

			fmt.Fprintln(out, tag)

			imageTags[imageName] = tag
		}
	}

	color.Default.Fprintln(out, "Tags generated in", time.Since(start))
	return imageTags, nil
}

// Stop stops the runner.
func (r *SkaffoldRunner) Stop() error {
	return r.RPCServerShutdown()
}
