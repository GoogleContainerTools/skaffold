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
	"encoding/json"
	"errors"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

func printAnalysis(out io.Writer, enableNewFormat bool, skipBuild bool, pairs []BuilderImagePair, unresolvedBuilderConfigs []InitBuilder, unresolvedImages []string) error {
	if !enableNewFormat {
		return PrintAnalyzeOldFormat(out, skipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
	}

	return PrintAnalyzeJSON(out, skipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
}

// TODO(nkubala): make these private again once DoInit() relinquishes control of the builder/image processing
func PrintAnalyzeOldFormat(out io.Writer, skipBuild bool, pairs []BuilderImagePair, unresolvedBuilders []InitBuilder, unresolvedImages []string) error {
	if !skipBuild && len(unresolvedBuilders) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again, or run `skaffold init --skip-build`")
	}

	a := struct {
		Dockerfiles []string `json:"dockerfiles,omitempty"`
		Images      []string `json:"images,omitempty"`
	}{Images: unresolvedImages}

	for _, pair := range pairs {
		if pair.Builder.Name() == docker.Name {
			a.Dockerfiles = append(a.Dockerfiles, pair.Builder.Path())
		}
		a.Images = append(a.Images, pair.ImageName)
	}
	for _, config := range unresolvedBuilders {
		if config.Name() == docker.Name {
			a.Dockerfiles = append(a.Dockerfiles, config.Path())
		}
	}

	return json.NewEncoder(out).Encode(a)
}

// printAnalyzeJSON takes the automatically resolved builder/image pairs, the unresolved images, and the unresolved builders, and generates
// a JSON string containing builder config information,
func PrintAnalyzeJSON(out io.Writer, skipBuild bool, pairs []BuilderImagePair, unresolvedBuilders []InitBuilder, unresolvedImages []string) error {
	if !skipBuild && len(unresolvedBuilders) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again, or run `skaffold init --skip-build`")
	}

	// Build JSON output. Example schema is below:
	// {
	//     "builders":[
	//         {
	//             "name":"Docker",
	//             "payload":"path/to/Dockerfile"
	//         },
	//         {
	//             "name":"Name of Builder",
	//             "payload": { // Payload structure may vary depending on builder type
	//                 "path":"path/to/builder.config",
	//                 "targetImage":"gcr.io/project/images",
	//                 ...
	//             }
	//         },
	//     ],
	//     "images":[
	//         {"name":"gcr.io/project/images", "foundMatch":"true"}, // No need to prompt for this image since its builder was automatically resolved
	//         {"name":"another/image", "foundMatch":"false"},
	//     ],
	// }
	//
	// "builders" is the list of builder configurations, and contains a builder name and a builder-specific payload
	// "images" contains an image name and a boolean that indicates whether a builder/image pair can be automatically resolved (true) or if it requires prompting (false)
	type Builder struct {
		Name    string      `json:"name,omitempty"`
		Payload InitBuilder `json:"payload"`
	}
	type Image struct {
		Name       string `json:"name"`
		FoundMatch bool   `json:"foundMatch"`
	}
	a := struct {
		Builders []Builder `json:"builders,omitempty"`
		Images   []Image   `json:"images,omitempty"`
	}{}

	for _, pair := range pairs {
		a.Builders = append(a.Builders, Builder{Name: pair.Builder.Name(), Payload: pair.Builder})
		a.Images = append(a.Images, Image{Name: pair.ImageName, FoundMatch: true})
	}
	for _, config := range unresolvedBuilders {
		a.Builders = append(a.Builders, Builder{Name: config.Name(), Payload: config})
	}
	for _, image := range unresolvedImages {
		a.Images = append(a.Images, Image{Name: image, FoundMatch: false})
	}

	return json.NewEncoder(out).Encode(a)
}
