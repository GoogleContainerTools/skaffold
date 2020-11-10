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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/generator"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type defaultBuildInitializer struct {
	builders                   []InitBuilder
	builderImagePairs          []BuilderImagePair
	generatedBuilderImagePairs []GeneratedBuilderImagePair
	unresolvedImages           []string
	skipBuild                  bool
	force                      bool
	enableNewFormat            bool
	resolveImages              bool
}

func (d *defaultBuildInitializer) ProcessImages(images []string) error {
	if len(d.builders) == 0 {
		return errors.NoBuilderErr{}
	}
	if d.skipBuild {
		return nil
	}

	// if we're in `analyze` mode, we want to match if we can, but not resolve
	d.matchBuildersToImages(images)
	if d.resolveImages {
		return d.resolveBuilderImages()
	}
	return nil
}

func (d *defaultBuildInitializer) BuildConfig() latest.BuildConfig {
	return latest.BuildConfig{
		Artifacts: Artifacts(d.builderImagePairs),
	}
}

func (d *defaultBuildInitializer) PrintAnalysis(out io.Writer) error {
	return printAnalysis(out, d.enableNewFormat, d.skipBuild, d.builderImagePairs, d.builders, d.unresolvedImages)
}

func (d *defaultBuildInitializer) GenerateManifests() (map[GeneratedBuilderImagePair][]byte, error) {
	generatedManifests := map[GeneratedBuilderImagePair][]byte{}
	for _, pair := range d.generatedBuilderImagePairs {
		manifest, err := generator.Generate(pair.ImageName)
		if err != nil {
			return nil, fmt.Errorf("generating kubernetes manifest: %w", err)
		}
		generatedManifests[pair] = manifest
		d.builderImagePairs = append(d.builderImagePairs, pair.BuilderImagePair)
	}
	d.generatedBuilderImagePairs = nil
	return generatedManifests, nil
}

// matchBuildersToImages takes a list of builders and images, checks if any of the builders' configured target
// images match an image in the image list, and returns a list of the matching builder/image pairs. Also
// separately returns the builder configs and images that didn't have any matches.
func (d *defaultBuildInitializer) matchBuildersToImages(images []string) {
	pairs, unresolvedBuilders, unresolvedImages := matchBuildersToImages(d.builders, images)
	d.builderImagePairs = pairs
	d.unresolvedImages = unresolvedImages
	d.builders = unresolvedBuilders
}
