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
	cliArtifacts    []string
	artifactInfos   []ArtifactInfo
	builders        []InitBuilder
	skipBuild       bool
	enableNewFormat bool
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

func (c *cliBuildInitializer) BuildConfig() (latest.BuildConfig, []*latest.PortForwardResource) {
	return latest.BuildConfig{
		Artifacts: Artifacts(c.artifactInfos),
	}, nil
}

func (c *cliBuildInitializer) PrintAnalysis(out io.Writer) error {
	return printAnalysis(out, c.enableNewFormat, c.skipBuild, c.artifactInfos, c.builders, nil)
}

func (c *cliBuildInitializer) GenerateManifests(io.Writer, bool) (map[GeneratedArtifactInfo][]byte, error) {
	return nil, nil
}

func (c *cliBuildInitializer) processCliArtifacts() error {
	pairs, err := processCliArtifacts(c.cliArtifacts)
	if err != nil {
		return err
	}
	c.artifactInfos = pairs
	return nil
}

func processCliArtifacts(cliArtifacts []string) ([]ArtifactInfo, error) {
	var artifactInfos []ArtifactInfo
	for _, artifact := range cliArtifacts {
		// Parses artifacts in 1 of 2 forms:
		// 1. JSON in the form of: {"builder":"Name of Builder","payload":{...},"image":"image.name","context":"artifact.context"}.
		//    The builder field is parsed first to determine the builder type, and the payload is parsed
		//    afterwards once the type is determined.
		// 2. Key-value pair: `path/to/Dockerfile=imageName` (deprecated, historical, Docker-only)
		a := struct {
			Name      string `json:"builder"`
			Image     string `json:"image"`
			Workspace string `json:"context"`
		}{}
		if err := json.Unmarshal([]byte(artifact), &a); err != nil {
			// Not JSON, use backwards compatible method
			parts := strings.Split(artifact, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed artifact provided: %s", artifact)
			}
			artifactInfos = append(artifactInfos, ArtifactInfo{
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
			info := ArtifactInfo{Builder: parsed.Payload, ImageName: a.Image, Workspace: a.Workspace}
			artifactInfos = append(artifactInfos, info)

		// FIXME: shouldn't use a human-readable name?
		case jib.PluginName(jib.JibGradle), jib.PluginName(jib.JibMaven):
			parsed := struct {
				Payload jib.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			parsed.Payload.BuilderName = a.Name
			info := ArtifactInfo{Builder: parsed.Payload, ImageName: a.Image, Workspace: a.Workspace}
			artifactInfos = append(artifactInfos, info)

		case buildpacks.Name:
			parsed := struct {
				Payload buildpacks.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			info := ArtifactInfo{Builder: parsed.Payload, ImageName: a.Image, Workspace: a.Workspace}
			artifactInfos = append(artifactInfos, info)

		default:
			return nil, fmt.Errorf("unknown builder type in CLI artifacts: %q", a.Name)
		}
	}
	return artifactInfos, nil
}
