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
	"github.com/buildpacks/pack"
	packcfg "github.com/buildpacks/pack/config"
	"github.com/buildpacks/pack/project"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
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

	// Evaluate Env Vars.
	env, err := env(a, b.mode, projectDescriptor)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate env variables: %w", err)
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

	builderImage, runImage, pullPolicy := resolveDependencyImages(artifact, b.artifacts, a.Dependencies, b.pushImages)

	if err := runPackBuildFunc(ctx, color.GetWriter(out), b.localDocker, pack.BuildOptions{
		AppPath:      workspace,
		Builder:      builderImage,
		RunImage:     runImage,
		Buildpacks:   buildpacks,
		Env:          env,
		Image:        latest,
		PullPolicy:   pullPolicy,
		TrustBuilder: artifact.TrustBuilder,
		// TODO(dgageot): Support project.toml include/exclude.
		// FileFilter: func(string) bool { return true },
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
	case lifecycle.CodeFailedDetect, lifecycle.CodeFailedDetectWithErrors:
		return "buildpacks could not determine application type"
	case lifecycle.CodeAnalyzeError:
		return "buildpacks failed analyzing metadata from previous builds"
	case lifecycle.CodeRestoreError:
		return "buildpacks failed to restoring cached layers"
	case lifecycle.CodeFailedBuildWithErrors, lifecycle.CodeBuildError:
		return "buildpacks failed to build image"
	case lifecycle.CodeExportError:
		return "buildpacks failed to save image and cache layers"
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
func resolveDependencyImages(artifact *latest.BuildpackArtifact, r ArtifactResolver, deps []*latest.ArtifactDependency, pushImages bool) (string, string, packcfg.PullPolicy) {
	builderImage, runImage := artifact.Builder, artifact.RunImage
	builderImageLocal, runImageLocal := false, false

	// We mimic pack's behaviour and always pull the images on first build
	// (tracked via images.AreAlreadyPulled()), but we never pull on
	// subsequent builds.  And if either the builder or run image are
	// dependent images then we do not pull and use PullIfNecessary.
	pullPolicy := packcfg.PullAlways

	var found bool
	for _, d := range deps {
		if builderImage == d.Alias {
			builderImage, found = r.GetImageTag(d.ImageName)
			if !found {
				logrus.Fatalf("failed to resolve build result for required artifact %q", d.ImageName)
			}
			builderImageLocal = true
		}
		if runImage == d.Alias {
			runImage, found = r.GetImageTag(d.ImageName)
			if !found {
				logrus.Fatalf("failed to resolve build result for required artifact %q", d.ImageName)
			}
			runImageLocal = true
		}
	}

	if builderImageLocal && runImageLocal {
		// if both builder and run image are built locally, there's nothing to pull.
		pullPolicy = packcfg.PullNever
	} else if builderImageLocal || runImageLocal {
		// if only one of builder or run image is built locally, we can enable remote image pull only if that image is also pushed to remote.
		pullPolicy = packcfg.PullIfNotPresent

		// if remote image pull is disabled then the image that is not fetched from the required artifacts might not be latest.
		if !pushImages && builderImageLocal {
			logrus.Warnln("Disabled remote image pull since builder image is built locally. Buildpacks run image may not be latest.")
		}
		if !pushImages && runImageLocal {
			logrus.Warnln("Disabled remote image pull since run image is built locally. Buildpacks builder image may not be latest.")
		}
	}

	// if remote pull is enabled ensure that same images aren't pulled twice.
	if pullPolicy == packcfg.PullAlways && images.AreAlreadyPulled(builderImage, runImage) {
		pullPolicy = packcfg.PullNever
	}

	return builderImage, runImage, pullPolicy
}
