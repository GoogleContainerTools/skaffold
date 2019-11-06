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
	"path/filepath"

	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func (b *BuildpackBuilder) build(ctx context.Context, out io.Writer, workspace string, artifact *latest.BuildpackArtifact, tag string) (string, error) {
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
	if err := b.pull(ctx, out, builderImage, artifact.ForcePull); err != nil {
		return "", err
	}

	runImage, err := b.findRunImage(ctx, artifact, builderImage)
	if err != nil {
		return "", err
	}
	logrus.Debugln("Run image", runImage)
	if err := b.pull(ctx, out, runImage, artifact.ForcePull); err != nil {
		return "", err
	}

	logrus.Debugln("Get dependencies")
	deps, err := GetDependencies(ctx, workspace, artifact)
	if err != nil {
		return "", err
	}

	var paths []string
	for _, dep := range deps {
		paths = append(paths, filepath.Join(workspace, dep))
	}

	copyWorkspace := func(ctx context.Context, container string) error {
		return b.localDocker.CopyToContainer(ctx, container, "/workspace", workspace, paths)
	}

	// These volumes store the state shared between build steps.
	// After the build, they are deleted.
	cacheID := util.RandomID()
	packWorkspace := volume(mount.TypeVolume, fmt.Sprintf("pack-%s.workspace", cacheID), "/workspace")
	layers := volume(mount.TypeVolume, fmt.Sprintf("pack-%s.layers", cacheID), "/layers")

	// These volumes are kept after the build and shared with all the builds.
	// They handle the caching of layer both for build time and run time.
	buildCache := volume(mount.TypeVolume, "pack-cache-skaffold.build", "/cache")
	launchCache := volume(mount.TypeVolume, "pack-cache-skaffold.launch", "/launch-cache")

	// Some steps need access to the Docker socket to load/save images.
	dockerSocket := volume(mount.TypeBind, "/var/run/docker.sock", "/var/run/docker.sock")

	defer func() {
		// Don't use ctx. It might have been cancelled by Ctrl-C
		if err := b.localDocker.VolumeRemove(context.Background(), packWorkspace.Source, true); err != nil {
			logrus.Warnf("unable to delete the docker volume [%s]", packWorkspace.Source)
		}
		if err := b.localDocker.VolumeRemove(context.Background(), layers.Source, true); err != nil {
			logrus.Warnf("unable to delete the docker volume [%s]", layers.Source)
		}
	}()

	if err := b.localDocker.ContainerRun(ctx, out,
		docker.ContainerRun{
			Image:       builderImage,
			Command:     []string{"/lifecycle/detector"},
			BeforeStart: copyWorkspace,
			Mounts:      []mount.Mount{packWorkspace, layers},
		}, docker.ContainerRun{
			Image:   builderImage,
			Command: []string{"sh", "-c", "/lifecycle/restorer -path /cache && /lifecycle/analyzer -daemon " + latest},
			User:    "root",
			Mounts:  []mount.Mount{packWorkspace, layers, buildCache, dockerSocket},
		}, docker.ContainerRun{
			Image:   builderImage,
			Command: []string{"/lifecycle/builder"},
			Mounts:  []mount.Mount{packWorkspace, layers},
		}, docker.ContainerRun{
			Image:   builderImage,
			Command: []string{"sh", "-c", "/lifecycle/exporter -daemon -image " + runImage + " -launch-cache /launch-cache " + latest + " && /lifecycle/cacher -path /cache"},
			User:    "root",
			Mounts:  []mount.Mount{packWorkspace, layers, launchCache, buildCache, dockerSocket},
		},
	); err != nil {
		return "", err
	}

	return latest, nil
}

func volume(mountType mount.Type, source, target string) mount.Mount {
	return mount.Mount{Type: mountType, Source: source, Target: target}
}

// pull makes sure the given image is pre-pulled.
func (b *BuildpackBuilder) pull(ctx context.Context, out io.Writer, image string, force bool) error {
	if force || !b.localDocker.ImageExists(ctx, image) {
		if err := b.localDocker.Pull(ctx, out, image); err != nil {
			return err
		}
	}
	return nil
}
