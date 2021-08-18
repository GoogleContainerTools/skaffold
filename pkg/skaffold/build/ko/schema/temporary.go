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

// Temporary home for schema changes.

package schema

// KoArtifact builds images using [ko](https://github.com/google/ko).
type KoArtifact struct {
	// BaseImage overrides the default ko base image (`gcr.io/distroless/static:nonroot`).
	// Corresponds to, and overrides, the `defaultBaseImage` in `.ko.yaml`.
	BaseImage string `yaml:"fromImage,omitempty"`

	// Dependencies are the file dependencies that Skaffold should watch for both rebuilding and file syncing for this artifact.
	Dependencies *KoDependencies `yaml:"dependencies,omitempty"`

	// Labels are key-value string pairs to add to the image config.
	// For example: `{"foo":"bar"}`.
	Labels map[string]string `yaml:"labels,omitempty"`

	// Platforms is the list of platforms to build images for. Each platform
	// is of the format `os[/arch[/variant]]`, e.g., `linux/amd64`.
	// Defaults to `all` to build for all platforms supported by the
	// base image.
	Platforms []string `yaml:"platforms,omitempty"`
}

// KoDependencies is used to specify dependencies for an artifact built by ko.
type KoDependencies struct {
	// Paths should be set to the file dependencies for this artifact, so that the skaffold file watcher knows when to rebuild and perform file synchronization.
	// Defaults to {"go.mod", "**.go"}.
	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	// Ignore specifies the paths that should be ignored by skaffold's file watcher.
	// If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both rebuilds and file synchronization.
	Ignore []string `yaml:"ignore,omitempty"`
}

// Artifact are the items that need to be built, along with the context in which
// they should be built.
type Artifact struct {
	// ImageName is the name of the image to be built.
	// For example: `gcr.io/k8s-skaffold/example`.
	ImageName string `yaml:"image,omitempty" yamltags:"required"`

	// Workspace is the directory containing the artifact's sources.
	// Defaults to `.`.
	Workspace string `yaml:"context,omitempty" skaffold:"filepath"`

	// ArtifactType describes how to build an artifact.
	ArtifactType `yaml:",inline"`

	// Dependencies describes build artifacts that this artifact depends on.
	Dependencies []*ArtifactDependency `yaml:"requires,omitempty"`
}

// ArtifactType describes how to build an artifact.
type ArtifactType struct {
	// KoArtifact builds images using [ko](https://github.com/google/ko).
	KoArtifact *KoArtifact `yaml:"-,omitempty" yamltags:"oneOf=artifact"`
}

// ArtifactDependency describes a specific build dependency for an artifact.
type ArtifactDependency struct {
	// ImageName is a reference to an artifact's image name.
	ImageName string `yaml:"image" yamltags:"required"`
	// Alias is a token that is replaced with the image reference in the builder definition files.
	// For example, the `docker` builder will use the alias as a build-arg key.
	// Defaults to the value of `image`.
	Alias string `yaml:"alias,omitempty"`
}
