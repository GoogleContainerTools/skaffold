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
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type cliBuildInitializer struct {
	cliArtifacts      []string
	builderImagePairs []BuilderImagePair
	builders          []InitBuilder
	skipBuild         bool
	enableNewFormat   bool
}

func (c *cliBuildInitializer) ProcessImages(images []string) error {
	if len(c.builders) == 0 && len(c.cliArtifacts) == 0 {
		return errors.NoBuilderErr{}
	}
	if err := c.processCliArtifacts(); err != nil {
		return fmt.Errorf("processing cli artifacts: %w", err)
	}
	return nil
}

func (c *cliBuildInitializer) BuildConfig() latest.BuildConfig {
	return latest.BuildConfig{
		Artifacts: Artifacts(c.builderImagePairs),
	}
}

func (c *cliBuildInitializer) PrintAnalysis(out io.Writer) error {
	return printAnalysis(out, c.enableNewFormat, c.skipBuild, c.builderImagePairs, c.builders, nil)
}

func (c *cliBuildInitializer) GenerateManifests() (map[GeneratedBuilderImagePair][]byte, error) {
	return nil, nil
}

func (c *cliBuildInitializer) processCliArtifacts() error {
	pairs, err := processCliArtifacts(c.cliArtifacts)
	if err != nil {
		return err
	}
	c.builderImagePairs = pairs
	return nil
}

func processCliArtifacts(cliArtifacts []string) ([]BuilderImagePair, error) {
	var pairs []BuilderImagePair
	for _, artifact := range cliArtifacts {
		// Parses JSON in the form of: {"builder":"Name of Builder","payload":{...},"image":"image.name"}.
		// The builder field is parsed first to determine the builder type, and the payload is parsed
		// afterwards once the type is determined.
		a := struct {
			Name  string `json:"builder"`
			Image string `json:"image"`
		}{}
		if err := json.Unmarshal([]byte(artifact), &a); err != nil {
			// Not JSON, use backwards compatible method
			parts := strings.Split(artifact, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed artifact provided: %s", artifact)
			}
			pairs = append(pairs, BuilderImagePair{
				Builder:   docker.ArtifactConfig{File: parts[0]},
				ImageName: parts[1],
			})
			continue
		}

		// Use builder type to parse payload
		switch a.Name {
		case docker.Name:
			parsed := struct {
				Payload docker.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := BuilderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		// FIXME: shouldn't use a human-readable name?
		case jib.PluginName(jib.JibGradle), jib.PluginName(jib.JibMaven):
			parsed := struct {
				Payload jib.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			parsed.Payload.BuilderName = a.Name
			pair := BuilderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		case buildpacks.Name:
			parsed := struct {
				Payload buildpacks.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := BuilderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		default:
			return nil, fmt.Errorf("unknown builder type in CLI artifacts: %q", a.Name)
		}
	}
	return pairs, nil
}
