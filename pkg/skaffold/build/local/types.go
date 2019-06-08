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

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	cfg *latest.LocalBuild

	localDocker        docker.LocalDaemon
	localCluster       bool
	pushImages         bool
	prune              bool
	skipTests          bool
	kubeContext        string
	builtImages        []string
	insecureRegistries map[string]bool
}

// external dependencies are wrapped
// into private functions for testability

var getLocalCluster = configutil.GetLocalCluster

var getLocalDocker = func(runCtx *runcontext.RunContext) (docker.LocalDaemon, error) {
	return docker.NewAPIClient(runCtx.Opts.Prune(), runCtx.InsecureRegistries)
}

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(runCtx *runcontext.RunContext) (*Builder, error) {
	localDocker, err := getLocalDocker(runCtx)
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}

	localCluster, err := getLocalCluster()
	if err != nil {
		return nil, errors.Wrap(err, "getting localCluster")
	}

	var pushImages bool
	if runCtx.Cfg.Build.LocalBuild.Push == nil {
		pushImages = !localCluster
		logrus.Debugf("push value not present, defaulting to %t because localCluster is %t", pushImages, localCluster)
	} else {
		pushImages = *runCtx.Cfg.Build.LocalBuild.Push
	}

	return &Builder{
		cfg:                runCtx.Cfg.Build.LocalBuild,
		kubeContext:        runCtx.KubeContext,
		localDocker:        localDocker,
		localCluster:       localCluster,
		pushImages:         pushImages,
		skipTests:          runCtx.Opts.SkipTests,
		prune:              runCtx.Opts.Prune(),
		insecureRegistries: runCtx.InsecureRegistries,
	}, nil
}

// Labels are labels specific to local builder.
func (b *Builder) Labels() map[string]string {
	labels := map[string]string{
		constants.Labels.Builder: "local",
	}

	v, err := b.localDocker.ServerVersion(context.Background())
	if err == nil {
		labels[constants.Labels.DockerAPIVersion] = fmt.Sprintf("%v", v.APIVersion)
	}

	return labels
}

// Prune uses the docker API client to remove all images built with Skaffold
func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return docker.Prune(ctx, out, b.builtImages, b.localDocker)
}
