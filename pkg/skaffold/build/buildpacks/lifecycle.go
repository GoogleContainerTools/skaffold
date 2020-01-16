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

package buildpacks

import (
	"context"
	"io"
	"strings"

	"github.com/buildpacks/pack"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	runPackBuildFunc = runPackBuild
)

func (b *Builder) build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	artifact := a.BuildpackArtifact
	workspace := a.Workspace

	// To improve caching, we always build the image with [:latest] tag
	// This way, the lifecycle is able to "bootstrap" from the previously built image.
	// The image will then be tagged as usual with the tag provided by the tag policy.
	parsed, err := docker.ParseReference(tag)
	if err != nil {
		return "", errors.Wrapf(err, "parsing tag %s", tag)
	}
	latest := parsed.BaseName + ":latest"

	builderImage := artifact.Builder
	logrus.Debugln("Builder image", builderImage)
	// If ForcePull is false: we pull the image only if it's not there already.
	// If ForcePull is true: we will let `pack` always pull.
	// Ideally, we add a `--pullIfNotPresent` option to upstream `pack`.
	if !artifact.ForcePull {
		if err := b.pull(ctx, out, builderImage); err != nil {
			return "", err
		}
	}

	runImage := artifact.RunImage
	if !artifact.ForcePull {
		// If ForcePull is false: we pull the image only if it's not there already.
		// If ForcePull is true: we will let `pack` always pull.
		// Ideally, we add a `--pullIfNotPresent` option to upstream `pack`.
		var err error
		runImage, err = b.findRunImage(ctx, artifact, builderImage)
		if err != nil {
			return "", err
		}
		logrus.Debugln("Run image", runImage)

		if err := b.pull(ctx, out, runImage); err != nil {
			return "", err
		}
	}

	logrus.Debugln("Evaluate env variables")
	env, err := misc.EvaluateEnv(artifact.Env)
	if err != nil {
		return "", errors.Wrap(err, "unable to evaluate env variables")
	}

	if err := runPackBuildFunc(ctx, out, pack.BuildOptions{
		AppPath:  workspace,
		Builder:  builderImage,
		RunImage: runImage,
		Env:      envMap(env),
		Image:    latest,
		NoPull:   !artifact.ForcePull,
	}); err != nil {
		return "", err
	}

	return latest, nil
}

func runPackBuild(ctx context.Context, out io.Writer, opts pack.BuildOptions) error {
	packClient, err := pack.NewClient(pack.WithLogger(NewLogger(out)))
	if err != nil {
		return errors.Wrap(err, "unable to create pack client")
	}

	return packClient.Build(ctx, opts)
}

func envMap(env []string) map[string]string {
	kv := make(map[string]string)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		kv[parts[0]] = parts[1]
	}

	return kv
}

// pull makes sure the given image is pre-pulled.
func (b *Builder) pull(ctx context.Context, out io.Writer, image string) error {
	if b.localDocker.ImageExists(ctx, image) {
		return nil
	}
	return b.localDocker.Pull(ctx, out, image)
}
