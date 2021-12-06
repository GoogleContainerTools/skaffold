/*
Copyright 2021 The Skaffold Authors

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

package ko

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/publish"
	"golang.org/x/tools/go/packages"

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// Build an artifact using ko, and either push it to an image registry, or
// sideload it to the local docker daemon.
// Build prints the image name to the out io.Writer and returns the image
// identifier. The image identifier is the tag or digest for pushed images, or
// the docker image ID for sideloaded images.
func (b *Builder) Build(ctx context.Context, out io.Writer, a *latestV2.Artifact, ref string) (string, error) {
	koBuilder, err := b.newKoBuilder(ctx, a)
	if err != nil {
		return "", fmt.Errorf("error creating ko builder: %w", err)
	}

	koPublisher, err := b.newKoPublisher(ref)
	if err != nil {
		return "", fmt.Errorf("error creating ko publisher: %w", err)
	}
	defer koPublisher.Close()

	imageRef, err := b.buildAndPublish(ctx, a, koBuilder, koPublisher)
	if err != nil {
		return "", fmt.Errorf("could not build and publish ko image %q: %w", a.ImageName, err)
	}
	fmt.Fprintln(out, imageRef.Name())

	return b.getImageIdentifier(ctx, imageRef, ref)
}

// buildAndPublish the image using the ko builder and publisher.
func (b *Builder) buildAndPublish(ctx context.Context, a *latestV2.Artifact, koBuilder build.Interface, koPublisher publish.Interface) (name.Reference, error) {
	importpath, err := getImportPath(a)
	if err != nil {
		return nil, fmt.Errorf("could not determine Go import path for ko image %q: %w", a.ImageName, err)
	}
	imageMap, err := b.publishImages(ctx, []string{importpath}, koPublisher, koBuilder)
	if err != nil {
		return nil, fmt.Errorf("failed to publish ko image: %w", err)
	}
	imageRef, exists := imageMap[importpath]
	if !exists {
		return nil, fmt.Errorf("no built image found for Go import path %q build images: %+v", importpath, imageMap)
	}
	return imageRef, nil
}

// getImportPath determines the Go import path that ko should build.
// The import path is returned with the `ko://` scheme prefix.
//
// If the image name from the Skaffold config has the prefix `ko://`, then
// treat the remainder of the string as the Go import path to build. This
// matches current ko behavior for working with Kubernetes resource files, and
// it will allow ko users to easily migrate to Skaffold without changing their
// Kubernetes YAML files. See https://github.com/google/ko#yaml-changes.
//
// If the image name does _not_ start with `ko://`, determine the Go import
// path of the image workspace directory.
func getImportPath(a *latestV2.Artifact) (string, error) {
	if strings.HasPrefix(a.ImageName, build.StrictScheme) {
		return a.ImageName, nil
	}
	baseDir := filepath.Join(a.Workspace, a.KoArtifact.Dir)
	target := a.KoArtifact.Main
	pkgConfig := &packages.Config{
		Mode: packages.NeedName,
		Dir:  baseDir,
	}
	pkgs, err := packages.Load(pkgConfig, target)
	if err != nil {
		return "", fmt.Errorf("could not determine import path from directory %q and target %q: %v", baseDir, target, err)
	}
	if len(pkgs) != 1 {
		return "", fmt.Errorf("expected exactly one main package for directory %q and target %q, got %d: %v", baseDir, target, len(pkgs), err)
	}
	return build.StrictScheme + pkgs[0].PkgPath, nil
}

// getImageIdentifier returns the image tag or digest for published images (`pushImages=true`),
// or the image ID from the local Docker daemon for sideloaded images (`pushImages=false`).
func (b *Builder) getImageIdentifier(ctx context.Context, imageRef name.Reference, ref string) (string, error) {
	if b.pushImages {
		return imageRef.Identifier(), nil
	}
	imageIdentifier, err := b.localDocker.ImageID(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("could not get imageID from local Docker Daemon for image %s: %+v", ref, err)
	}
	return imageIdentifier, nil
}
