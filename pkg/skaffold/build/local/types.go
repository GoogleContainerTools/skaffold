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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
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
	devMode            bool
	kubeContext        string
	builtImages        []string
	insecureRegistries map[string]bool
}

// external dependencies are wrapped
// into private functions for testability

var getLocalCluster = config.GetLocalCluster

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(runCtx *runcontext.RunContext) (*Builder, error) {
	localDocker, err := docker.NewAPIClient(runCtx)
	if err != nil {
		return nil, fmt.Errorf("getting docker client: %w", err)
	}

	// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
	// remove minikubeProfile from here and instead detect it by matching the
	// kubecontext API Server to minikube profiles

	localCluster, err := getLocalCluster(runCtx.Opts.GlobalConfig, runCtx.Opts.MinikubeProfile)
	if err != nil {
		return nil, fmt.Errorf("getting localCluster: %w", err)
	}

	var pushImages bool
	if runCtx.Cfg.Build.LocalBuild.Push == nil {
		pushImages = !localCluster
		logrus.Debugf("push value not present, defaulting to %t because localCluster is %t", pushImages, localCluster)
	} else {
		pushImages = *runCtx.Cfg.Build.LocalBuild.Push
	}

	return &Builder{
		cfg:                *runCtx.Cfg.Build.LocalBuild,
		kubeContext:        runCtx.KubeContext,
		localDocker:        localDocker,
		localCluster:       localCluster,
		pushImages:         pushImages,
		skipTests:          runCtx.Opts.SkipTests,
		devMode:            runCtx.Opts.IsDevMode(),
		prune:              runCtx.Opts.Prune(),
		pruneChildren:      !runCtx.Opts.NoPruneChildren,
		insecureRegistries: runCtx.InsecureRegistries,
	}, nil
}

func (b *Builder) PushImages() bool {
	return b.pushImages
}

// Prune uses the docker API client to remove all images built with Skaffold
func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return b.localDocker.Prune(ctx, out, b.builtImages, b.pruneChildren)
}
