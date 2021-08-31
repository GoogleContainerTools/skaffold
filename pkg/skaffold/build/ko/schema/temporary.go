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

	// Dir is the directory where the `go` tool will be run.
	// The value is a directory path relative to the `context` directory.
	// If empty, the `go` tool will run in the `context` directory.
	// Example: `./my-app-sources`
	Dir string `yaml:"dir,omitempty"`

	// Env are environment variables, in the `key=value` form, passed to the build.
	// These environment variables are only used at build time.
	// They are _not_ set in the resulting container image.
	// For example: `["GOPRIVATE=source.developers.google.com", "GOCACHE=/workspace/.gocache"]`.
	Env []string `yaml:"env,omitempty"`

	// Flags are additional build flags passed to the builder.
	// For example: `["-trimpath", "-v"]`.
	Flags []string `yaml:"args,omitempty"`

	// Labels are key-value string pairs to add to the image config.
	// For example: `{"org.opencontainers.image.source":"https://github.com/GoogleContainerTools/skaffold"}`.
	Labels map[string]string `yaml:"labels,omitempty"`

	// Ldflags are linker flags passed to the builder.
	// For example: `["-buildid=", "-s", "-w"]`.
	Ldflags []string `yaml:"ldflags,omitempty"`

	// Platforms is the list of platforms to build images for. Each platform
	// is of the format `os[/arch[/variant]]`, e.g., `linux/amd64`.
	// Defaults to `all` to build for all platforms supported by the
	// base image.
	Platforms []string `yaml:"platforms,omitempty"`

	// Target is the location of the main package.
	// If target is specified as a relative path, it is relative to the `context` directory.
	// If target is empty, the ko builder looks for the main package in the `context` directory only, but not in any subdirectories.
	// If target is a pattern with wildcards, such as `./...`, the expansion must contain only one main package, otherwise ko fails.
	// Target is ignored if the `ImageName` starts with `ko://`.
	// Example: `./cmd/foo`
	Target string `yaml:"target,omitempty"`
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
