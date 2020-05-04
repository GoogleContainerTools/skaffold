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
	"fmt"
	"io"
	"strings"

	"github.com/buildpacks/pack"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	runPackBuildFunc = runPackBuild
)

// images is a global list of builder/runner image pairs that are already pulled.
// In a skaffold session, typically a skaffold dev loop, we want to avoid asking `pack`
// to pull the images that are already pulled.
var images pulledImages

func (b *Builder) build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	artifact := a.BuildpackArtifact
	workspace := a.Workspace

	// To improve caching, we always build the image with [:latest] tag
	// This way, the lifecycle is able to "bootstrap" from the previously built image.
	// The image will then be tagged as usual with the tag provided by the tag policy.
	parsed, err := docker.ParseReference(tag)
	if err != nil {
		return "", fmt.Errorf("parsing tag %q: %w", tag, err)
	}
	latest := parsed.BaseName + ":latest"

	logrus.Debugln("Evaluate env variables")
	env, err := misc.EvaluateEnv(artifact.Env)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate env variables: %w", err)
	}

	if b.devMode && a.Sync != nil && a.Sync.Auto != nil {
		env = append(env, "GOOGLE_DEVMODE=1")
	}

	alreadyPulled := images.AreAlreadyPulled(artifact.Builder, artifact.RunImage)

	if err := runPackBuildFunc(ctx, out, b.localDocker, pack.BuildOptions{
		AppPath:    workspace,
		Builder:    artifact.Builder,
		RunImage:   artifact.RunImage,
		Buildpacks: artifact.Buildpacks,
		Env:        envMap(env),
		Image:      latest,
		NoPull:     alreadyPulled,
	}); err != nil {
		return "", err
	}

	images.MarkAsPulled(artifact.Builder, artifact.RunImage)

	return latest, nil
}

func runPackBuild(ctx context.Context, out io.Writer, localDocker docker.LocalDaemon, opts pack.BuildOptions) error {
	packClient, err := pack.NewClient(
		pack.WithDockerClient(localDocker.RawClient()),
		pack.WithLogger(NewLogger(out)),
	)
	if err != nil {
		return fmt.Errorf("unable to create pack client: %w", err)
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
