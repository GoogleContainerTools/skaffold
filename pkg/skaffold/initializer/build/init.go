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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/generator"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

type defaultBuildInitializer struct {
	builders               []InitBuilder
	artifactInfos          []ArtifactInfo
	generatedArtifactInfos []GeneratedArtifactInfo
	manifests              []*generator.Container
	unresolvedImages       []string
	skipBuild              bool
	force                  bool
	enableNewFormat        bool
	resolveImages          bool
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

func (d *defaultBuildInitializer) BuildConfig() (latest.BuildConfig, []*latest.PortForwardResource) {
	pf := []*latest.PortForwardResource{}

	for _, manifestInfo := range d.manifests {
		// Port value is set to 0 if user decides to not port forward service
		if manifestInfo.Port != 0 {
			pf = append(pf, &latest.PortForwardResource{
				Type: "service",
				Name: manifestInfo.Name,
				Port: util.FromInt(manifestInfo.Port),
			})
		}
	}

	return latest.BuildConfig{
		Artifacts: Artifacts(d.artifactInfos),
	}, pf
}

func (d *defaultBuildInitializer) PrintAnalysis(out io.Writer) error {
	return printAnalysis(out, d.enableNewFormat, d.skipBuild, d.artifactInfos, d.builders, d.unresolvedImages)
}

func (d *defaultBuildInitializer) GenerateManifests(out io.Writer, force bool) (map[GeneratedArtifactInfo][]byte, error) {
	generatedManifests := map[GeneratedArtifactInfo][]byte{}
	for _, info := range d.generatedArtifactInfos {
		port := 8080
		var err error
		if !force {
			port, err = prompt.PortForwardResourceFunc(out, info.ImageName)
			if err != nil {
				return nil, fmt.Errorf("getting port input: %w", err)
			}
		}

		manifest, manifestInfo, err := generator.Generate(info.ImageName, port)
		if err != nil {
			return nil, fmt.Errorf("generating kubernetes manifest: %w", err)
		}
		generatedManifests[info] = manifest
		d.manifests = append(d.manifests, manifestInfo)
		d.artifactInfos = append(d.artifactInfos, info.ArtifactInfo)
	}
	d.generatedArtifactInfos = nil
	return generatedManifests, nil
}

// matchBuildersToImages takes a list of builders and images, checks if any of the builders' configured target
// images match an image in the image list, and returns a list of the matching builder/image pairs. Also
// separately returns the builder configs and images that didn't have any matches.
func (d *defaultBuildInitializer) matchBuildersToImages(images []string) {
	artifactInfos, unresolvedBuilders, unresolvedImages := matchBuildersToImages(d.builders, images)
	d.artifactInfos = artifactInfos
	d.unresolvedImages = unresolvedImages
	d.builders = unresolvedBuilders
}
