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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const PollInterval = 500 * time.Millisecond

// SkaffoldRunner is responsible for running the skaffold build and deploy pipeline.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer

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

	builder, err := getBuilder(tagger, &cfg.Build, kubeContext)
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
		watchFactory: watch.NewCompositeWatcher,
	}, nil
}

func getBuilder(t tag.Tagger, cfg *v1alpha2.BuildConfig, kubeContext string) (build.Builder, error) {
	switch {
	case cfg.LocalBuild != nil:
		logrus.Debugf("Using builder: local")
		return build.NewLocalBuilder(t, cfg.LocalBuild, kubeContext)

	case cfg.GoogleCloudBuild != nil:
		logrus.Debugf("Using builder: google cloud")
		return build.NewGoogleCloudBuilder(t, cfg.GoogleCloudBuild), nil

	case cfg.KanikoBuild != nil:
		logrus.Debugf("Using builder: kaniko")
		return build.NewKanikoBuilder(t, cfg.KanikoBuild), nil

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
		return deploy.NewKubectlDeployer(cwd, cfg.KubectlDeploy, kubeContext), nil

	case cfg.HelmDeploy != nil:
		return deploy.NewHelmDeployer(cfg.HelmDeploy, kubeContext, namespace), nil

	case cfg.KustomizeDeploy != nil:
		return deploy.NewKustomizeDeployer(cfg.KustomizeDeploy, kubeContext), nil

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

// Run builds artifacts ad then deploys them.
func (r *SkaffoldRunner) Run(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact) error {
	bRes, err := r.Build(ctx, out, artifacts)
	if err != nil {
		return errors.Wrap(err, "build step")
	}

	_, err = r.Deploy(ctx, out, bRes)
	if err != nil {
		return errors.Wrap(err, "deploy step")
	}

	return nil
}

// Dev watches for changes and runs the skaffold build and deploy
// pipeline until interrrupted by the user.
func (r *SkaffoldRunner) Dev(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact) ([]build.Artifact, error) {
	imageList := kubernetes.NewImageList()
	colorPicker := kubernetes.NewColorPicker(artifacts)
	logger := kubernetes.NewLogAggregator(out, imageList, colorPicker)

	deployDeps, err := r.Dependencies()
	if err != nil {
		return nil, errors.Wrap(err, "getting deploy dependencies")
	}
	logrus.Infof("Deployer dependencies: %s", deployDeps)

	onDeploymentsChange := func(changedPaths []string) error {
		logger.Mute()
		_, err := r.Deploy(ctx, out, r.builds)
		if err != nil {
			logrus.Warnln("Skipping Deploy due to error:", err)
		}
		logger.Unmute()

		fmt.Fprintln(out, "Watching for changes...")
		return nil
	}

	onArtifactChange := func(changes []*v1alpha2.Artifact) error {
		logger.Mute()
		err := r.buildAndDeploy(ctx, out, changes, imageList)
		logger.Unmute()

		fmt.Fprintln(out, "Watching for changes...")
		return err
	}

	if err := onArtifactChange(artifacts); err != nil {
		return nil, errors.Wrap(err, "first run")
	}

	// Start logs
	if err = logger.Start(ctx); err != nil {
		return nil, errors.Wrap(err, "starting logger")
	}

	watcher := r.watchFactory(deployDeps, artifacts, PollInterval)
	return nil, watcher.Run(ctx, onDeploymentsChange, onArtifactChange)
}

// buildAndDeploy builds a subset of the artifacts and deploys everything.
func (r *SkaffoldRunner) buildAndDeploy(ctx context.Context, out io.Writer, artifacts []*v1alpha2.Artifact, images *kubernetes.ImageList) error {
	firstRun := r.builds == nil

	bRes, err := r.Build(ctx, out, artifacts)
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
