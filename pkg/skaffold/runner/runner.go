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
	"io"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
)

// SkaffoldRunner is responsible for running the skaffold build and deploy pipeline.
type SkaffoldRunner struct {
	build.Builder
	deploy.Deployer
	tag.Tagger
	watch.Watcher

	devMode bool

	config     *config.SkaffoldConfig
	watchReady chan *watch.Event
	cancel     chan struct{}

	out io.Writer
}

// NewForConfig returns a new SkaffoldRunner for a SkaffoldConfig
func NewForConfig(out io.Writer, dev bool, cfg *config.SkaffoldConfig) (*SkaffoldRunner, error) {
	builder, err := getBuilder(&cfg.Build)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold build config")
	}
	deployer, err := getDeployer(&cfg.Deploy)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold deploy config")
	}
	tagger, err := newTaggerForConfig(cfg.Build.TagPolicy)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold tag config")
	}
	return &SkaffoldRunner{
		config:   cfg,
		Builder:  builder,
		Deployer: deployer,
		Tagger:   tagger,
		Watcher:  &watch.FSWatcher{}, //TODO(@r2d4): should this be configurable?
		devMode:  dev,
		cancel:   make(chan struct{}, 1),
		out:      out,
	}, nil
}

func getBuilder(cfg *config.BuildConfig) (build.Builder, error) {
	if cfg != nil && cfg.LocalBuild != nil {
		return build.NewLocalBuilder(cfg)
	}
	return nil, fmt.Errorf("Unknown builder for config %+v", cfg)
}

func getDeployer(cfg *config.DeployConfig) (deploy.Deployer, error) {
	if cfg.KubectlDeploy != nil {
		return deploy.NewKubectlDeployer(cfg)
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
	if r.devMode {
		return r.dev()
	}
	return r.run()
}

func (r *SkaffoldRunner) dev() error {
	for {
		if err := r.run(); err != nil {
			return errors.Wrap(err, "running build and deploy")
		}
		evt, err := r.Watch(r.config.Build.Artifacts, r.watchReady, r.cancel)
		if err != nil {
			return errors.Wrap(err, "running watch")
		}
		if evt.EventType == watch.WatchStop {
			return nil
		}
	}
}

func (r *SkaffoldRunner) run() error {
	res, err := r.Builder.Run(r.out, r.Tagger)
	if err != nil {
		return errors.Wrap(err, "build step")
	}

	if _, err := r.Deployer.Run(res); err != nil {
		return errors.Wrap(err, "deploy step")
	}
	return nil
}
