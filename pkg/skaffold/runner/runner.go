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
	"fmt"

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
	watch.Watcher

	opts       *config.SkaffoldOptions
	config     *config.SkaffoldConfig
	watchReady chan *watch.Event
	cancel     chan struct{}
	kubeclient clientgo.Interface
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
		config:     cfg,
		Builder:    builder,
		Deployer:   deployer,
		Tagger:     tagger,
		Watcher:    &watch.FSWatcher{}, //TODO(@r2d4): should this be configurable?
		opts:       opts,
		kubeclient: client,
		cancel:     make(chan struct{}, 1),
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
		return r.dev()
	}

	if _, _, err := r.run(r.config.Build.Artifacts); err != nil {
		return errors.Wrap(err, "run")
	}
	return nil
}

func (r *SkaffoldRunner) dev() error {
	// First rebuild everything.
	bRes, _, err := r.run(r.config.Build.Artifacts)
	if err != nil {
		// In dev mode, we only warn on pipeline errors
		logrus.Warnf("run: %s", err)
	}
	for {
		if bRes != nil {
			for _, b := range bRes.Builds {
				b := b
				go kubernetes.StreamLogsRetry(r.opts.Output, r.kubeclient.CoreV1(), b.Tag, b.Digest, 5)
			}
		}
		evt, err := r.Watch(r.config.Build.Artifacts, r.watchReady, r.cancel)
		if err != nil {
			return errors.Wrap(err, "running watch")
		}
		if evt.EventType == watch.WatchStop {
			return nil
		}
		bRes, _, err = r.run(evt.ChangedArtifacts)
		if err != nil {
			// In dev mode, we only warn on pipeline errors
			logrus.Warnf("run: %s", err)
		}
	}
}

func (r *SkaffoldRunner) run(artifacts []*config.Artifact) (*build.BuildResult, *deploy.Result, error) {
	fmt.Fprint(r.opts.Output, "Starting build...\n")
	bRes, err := r.Builder.Run(r.opts.Output, r.Tagger, artifacts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "build step")
	}
	fmt.Fprint(r.opts.Output, "Build complete.\n")

	fmt.Fprint(r.opts.Output, "Starting deploy...\n")
	if _, err := r.Deployer.Run(r.opts.Output, bRes); err != nil {
		return nil, nil, errors.Wrap(err, "deploy step")
	}
	if r.opts.Notification {
		fmt.Fprint(r.opts.Output, constants.TerminalBell)
	}
	fmt.Fprint(r.opts.Output, "Deploy complete.\n")
	return bRes, nil, nil
}
