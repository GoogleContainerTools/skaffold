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
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	ValidateConfig = validateConfig
)

// Name is the name of the Buildpack builder
var Name = "Buildpacks"

// Buildpack holds information about a Buildpack project
type Buildpacks struct {
	File string `json:"path,omitempty"`
}

// Name returns the name of the builder
func (b Buildpacks) Name() string {
	return Name
}

// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
func (b Buildpacks) Describe() string {
	return fmt.Sprintf("%s (%s)", b.Name(), b.File)
}

// CreateArtifact creates an Artifact to be included in the generated Build Config
func (b Buildpacks) UpdateArtifact(a *latest.Artifact) {
	a.ArtifactType = latest.ArtifactType{
		BuildpackArtifact: &latest.BuildpackArtifact{
			Builder: "heroku/buildpacks",
		},
	}
}

// ConfiguredImage returns the target image configured by the builder, or empty string if no image is configured
func (b Buildpacks) ConfiguredImage() string {
	// Target image is not configured in dockerfiles
	return ""
}

// Path returns the path to the build definition
func (b Buildpacks) Path() string {
	return b.File
}

// validateConfig checks if a file is a valid Buildpack configuration.
func validateConfig(path string) bool {
	return filepath.Base(path) == "package.json"
}
