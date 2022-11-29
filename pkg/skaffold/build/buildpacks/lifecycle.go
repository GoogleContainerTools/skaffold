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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	lifecycle "github.com/buildpacks/lifecycle/cmd"
	pack "github.com/buildpacks/pack/pkg/client"
	packimg "github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/project"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
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
	clearCache := artifact.ClearCache
	workspace := a.Workspace

	// Read `project.toml` if it exists.
	path := filepath.Join(workspace, artifact.ProjectDescriptor)
	projectDescriptor, err := project.ReadProjectDescriptor(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read project descriptor %q: %w", path, err)
	}

	// To improve caching, we always build the image with [:latest] tag
	// This way, the lifecycle is able to "bootstrap" from the previously built image.
	// The image will then be tagged as usual with the tag provided by the tag policy.
	parsed, err := docker.ParseReference(tag)
	if err != nil {
		return "", fmt.Errorf("parsing tag %q: %w", tag, err)
	}
	latest := parsed.BaseName + ":latest"

	// Evaluate Env Vars replacing those in project.toml.
	env, err := env(a, b.mode, projectDescriptor)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate env variables: %w", err)
	}
	projectDescriptor.Build.Env = nil

	cc, err := containerConfig(artifact)
	if err != nil {
		return "", fmt.Errorf("%q: %w", a.ImageName, err)
	}

	// List buildpacks to be used for the build.
	// Those specified in the skaffold.yaml replace those in the project.toml.
	buildpacks := artifact.Buildpacks
	if len(buildpacks) == 0 {
		for _, bp := range projectDescriptor.Build.Buildpacks {
			if bp.ID != "" {
				if bp.Version == "" {
					buildpacks = append(buildpacks, bp.ID)
				} else {
					buildpacks = append(buildpacks, fmt.Sprintf("%s@%s", bp.ID, bp.Version))
				}
				// } else {
				// TODO(dgageot): Support URI.
			}
		}
	}
	projectDescriptor.Build.Buildpacks = nil

	builderImage, runImage, pullPolicy := resolveDependencyImages(artifact, b.artifacts, a.Dependencies, b.pushImages)

	if err := runPackBuildFunc(ctx, out, b.localDocker, pack.BuildOptions{
		AppPath:           workspace,
		Builder:           builderImage,
		RunImage:          runImage,
		Buildpacks:        buildpacks,
		Env:               env,
		Image:             latest,
		PullPolicy:        pullPolicy,
		TrustBuilder:      func(_ string) bool { return artifact.TrustBuilder },
		ClearCache:        clearCache,
		ContainerConfig:   cc,
		ProjectDescriptor: projectDescriptor,
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
		pack.WithFetcher(newFetcher(out, localDocker)),
	)
	if err != nil {
		return fmt.Errorf("unable to create pack client: %w", err)
	}

	err = packClient.Build(ctx, opts)
	// pack turns exit codes from the lifecycle into `failed with status code: N`
	if err != nil {
		err = rewriteLifecycleStatusCode(err)
	}
	return err
}

func rewriteLifecycleStatusCode(lce error) error {
	prefix := "failed with status code: "
	lceText := lce.Error()
	if strings.HasPrefix(lceText, prefix) {
		sc := lceText[len(prefix):]
		if code, err := strconv.Atoi(sc); err == nil {
			return errors.New(mapLifecycleStatusCode(code))
		}
	}
	return lce
}

func mapLifecycleStatusCode(code int) string {
	switch code {
	case lifecycle.CodeFailed:
		return "buildpacks lifecycle failed"
	case lifecycle.CodeInvalidArgs:
		return "lifecycle reported invalid arguments"
	case lifecycle.CodeIncompatiblePlatformAPI:
		return "incompatible version of Platform API"
	case lifecycle.CodeIncompatibleBuildpackAPI:
		return "incompatible version of Buildpacks API"
	default:
		// we should never see CodeRebaseError or CodeLaunchError
		return fmt.Sprintf("lifecycle failed with status code %d", code)
	}
}

func envMap(env []string) map[string]string {
	kv := make(map[string]string)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		kv[parts[0]] = parts[1]
	}

	return kv
}

// resolveDependencyImages replaces the provided builder and run images with built images from the required artifacts if specified.
// The return values are builder image, run image, and if remote pull is required.
func resolveDependencyImages(artifact *latest.BuildpackArtifact, r ArtifactResolver, deps []*latest.ArtifactDependency, pushImages bool) (string, string, packimg.PullPolicy) {
	builderImage, runImage := artifact.Builder, artifact.RunImage
	builderImageLocal, runImageLocal := false, false

	// We mimic pack's behaviour and always pull the images on first build
	// (tracked via images.AreAlreadyPulled()), but we never pull on
	// subsequent builds.  And if either the builder or run image are
	// dependent images then we do not pull and use PullIfNecessary.
	pullPolicy := packimg.PullAlways

	var found bool
	for _, d := range deps {
		if builderImage == d.Alias {
			builderImage, found = r.GetImageTag(d.ImageName)
			if !found {
				log.Entry(context.TODO()).Fatalf("failed to resolve build result for required artifact %q", d.ImageName)
			}
			builderImageLocal = true
		}
		if runImage == d.Alias {
			runImage, found = r.GetImageTag(d.ImageName)
			if !found {
				log.Entry(context.TODO()).Fatalf("failed to resolve build result for required artifact %q", d.ImageName)
			}
			runImageLocal = true
		}
	}

	if builderImageLocal && runImageLocal {
		// if both builder and run image are built locally, there's nothing to pull.
		pullPolicy = packimg.PullNever
	} else if builderImageLocal || runImageLocal {
		// if only one of builder or run image is built locally, we can enable remote image pull only if that image is also pushed to remote.
		pullPolicy = packimg.PullIfNotPresent

		// if remote image pull is disabled then the image that is not fetched from the required artifacts might not be latest.
		if !pushImages && builderImageLocal {
			log.Entry(context.TODO()).Warn("Disabled remote image pull since builder image is built locally. Buildpacks run image may not be latest.")
		}
		if !pushImages && runImageLocal {
			log.Entry(context.TODO()).Warn("Disabled remote image pull since run image is built locally. Buildpacks builder image may not be latest.")
		}
	}

	// if remote pull is enabled ensure that same images aren't pulled twice.
	if pullPolicy == packimg.PullAlways && images.AreAlreadyPulled(builderImage, runImage) {
		pullPolicy = packimg.PullNever
	}

	return builderImage, runImage, pullPolicy
}

func containerConfig(artifact *latest.BuildpackArtifact) (pack.ContainerConfig, error) {
	var vols []string
	for _, v := range artifact.Volumes {
		if v.Host == "" || v.Target == "" {
			// in case these slip by the JSON schema
			return pack.ContainerConfig{}, errors.New("buildpacks volumes must have both host and target")
		}
		var spec string
		if v.Options == "" {
			spec = fmt.Sprintf("%s:%s", v.Host, v.Target)
		} else {
			spec = fmt.Sprintf("%s:%s:%s", v.Host, v.Target, v.Options)
		}
		vols = append(vols, spec)
	}

	return pack.ContainerConfig{Volumes: vols}, nil
}
