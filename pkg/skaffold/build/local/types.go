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

package local

import (
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	cfg latest.LocalBuild

	localDocker        docker.LocalDaemon
	localCluster       bool
	pushImages         bool
	prune              bool
	pruneChildren      bool
	skipTests          bool
	mode               config.RunMode
	kubeContext        string
	builtImages        []string
	insecureRegistries map[string]bool
	muted              build.Muted
}

// external dependencies are wrapped
// into private functions for testability

var getLocalCluster = config.GetLocalCluster

type Config interface {
	docker.Config

	Pipeline() latest.Pipeline
	GlobalConfig() string
	GetKubeContext() string
	DetectMinikube() bool
	SkipTests() bool
	Mode() config.RunMode
	NoPruneChildren() bool
	Muted() config.Muted
}

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(cfg Config) (*Builder, error) {
	localDocker, err := docker.NewAPIClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("getting docker client: %w", err)
	}

	// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
	// remove minikubeProfile from here and instead detect it by matching the
	// kubecontext API Server to minikube profiles

	localCluster, err := getLocalCluster(cfg.GlobalConfig(), cfg.MinikubeProfile(), cfg.DetectMinikube())
	if err != nil {
		return nil, fmt.Errorf("getting localCluster: %w", err)
	}

	var pushImages bool
	if cfg.Pipeline().Build.LocalBuild.Push == nil {
		pushImages = !localCluster
		logrus.Debugf("push value not present, defaulting to %t because localCluster is %t", pushImages, localCluster)
	} else {
		pushImages = *cfg.Pipeline().Build.LocalBuild.Push
	}

	return &Builder{
		cfg:                *cfg.Pipeline().Build.LocalBuild,
		kubeContext:        cfg.GetKubeContext(),
		localDocker:        localDocker,
		localCluster:       localCluster,
		pushImages:         pushImages,
		skipTests:          cfg.SkipTests(),
		mode:               cfg.Mode(),
		prune:              cfg.Prune(),
		pruneChildren:      !cfg.NoPruneChildren(),
		insecureRegistries: cfg.GetInsecureRegistries(),
		muted:              cfg.Muted(),
	}, nil
}

func (b *Builder) PushImages() bool {
	return b.pushImages
}

// Prune uses the docker API client to remove all images built with Skaffold
func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return b.localDocker.Prune(ctx, out, b.builtImages, b.pruneChildren)
}
