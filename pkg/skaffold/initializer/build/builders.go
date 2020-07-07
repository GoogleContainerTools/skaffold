/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NoBuilder allows users to specify they don't want to build
// an image we parse out from a Kubernetes manifest
const NoBuilder = "None (image not built from these sources)"

// InitBuilder represents a builder that can be chosen by skaffold init.
type InitBuilder interface {
	// Name returns the name of the builder.
	Name() string

	// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
	// Must be unique between artifacts.
	Describe() string

	// ArtifactType returns the type of the artifact to be built.
	ArtifactType() latest.ArtifactType

	// ConfiguredImage returns the target image configured by the builder, or an empty string if no image is configured.
	// This should be a cheap operation.
	ConfiguredImage() string

	// Path returns the path to the build file
	Path() string
}

// BuilderImagePair defines a builder and the image it builds
type BuilderImagePair struct {
	Builder   InitBuilder
	ImageName string
}

// GeneratedBuilderImagePair pairs a discovered builder with a
// generated image name, and the path to the manifest that should be generated
type GeneratedBuilderImagePair struct {
	BuilderImagePair
	ManifestPath string
}

type Initializer interface {
	// ProcessImages is the entrypoint call, and handles the pairing of all builders
	// contained in the initializer with the provided images from the deploy initializer
	ProcessImages([]string) error
	// BuildConfig returns the processed build config to be written to the skaffold.yaml
	BuildConfig() latest.BuildConfig
	// PrintAnalysis writes the project analysis to the provided out stream
	PrintAnalysis(io.Writer) error
	// GenerateManifests generates image names and manifests for all unresolved pairs
	GenerateManifests() (map[GeneratedBuilderImagePair][]byte, error)
}

type emptyBuildInitializer struct {
}

func (e *emptyBuildInitializer) ProcessImages([]string) error {
	return nil
}

func (e *emptyBuildInitializer) BuildConfig() latest.BuildConfig {
	return latest.BuildConfig{}
}

func (e *emptyBuildInitializer) PrintAnalysis(io.Writer) error {
	return nil
}

func (e *emptyBuildInitializer) GenerateManifests() (map[GeneratedBuilderImagePair][]byte, error) {
	return nil, nil
}

func NewInitializer(builders []InitBuilder, c config.Config) Initializer {
	switch {
	case c.SkipBuild:
		return &emptyBuildInitializer{}
	case c.CliArtifacts != nil:
		return &cliBuildInitializer{
			cliArtifacts:    c.CliArtifacts,
			builders:        builders,
			skipBuild:       c.SkipBuild,
			enableNewFormat: c.EnableNewInitFormat,
		}
	default:
		return &defaultBuildInitializer{
			builders:        builders,
			skipBuild:       c.SkipBuild,
			force:           c.Force,
			enableNewFormat: c.EnableNewInitFormat,
			resolveImages:   !c.Analyze,
		}
	}
}
