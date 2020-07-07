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
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	Validate = validate
)

// Name is the name of the Buildpack builder
var Name = "Buildpacks"

// ArtifactConfig holds information about a Buildpack project
type ArtifactConfig struct {
	File    string `json:"path,omitempty"`
	Builder string `json:"builder,omitempty"`
}

// Name returns the name of the builder
func (c ArtifactConfig) Name() string {
	return Name
}

// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
func (c ArtifactConfig) Describe() string {
	return fmt.Sprintf("%s (%s)", c.Name(), c.File)
}

// ArtifactType returns the type of the artifact to be built.
func (c ArtifactConfig) ArtifactType() latest.ArtifactType {
	return latest.ArtifactType{
		BuildpackArtifact: &latest.BuildpackArtifact{
			Builder: c.Builder,
		},
	}
}

// ConfiguredImage returns the target image configured by the builder, or empty string if no image is configured
func (c ArtifactConfig) ConfiguredImage() string {
	// Target image is not configured in buildpacks
	return ""
}

// Path returns the path to the build definition
func (c ArtifactConfig) Path() string {
	return c.File
}

// validate checks if a file is a valid Buildpack configuration.
func validate(path string) bool {
	switch filepath.Base(path) {
	// Buildpacks project descriptor.
	case "project.toml":
		return true

	// NodeJS.
	case "package.json":
		return true

	// Go.
	case "go.mod":
		return true

	// Java.
	case "pom.xml", "build.gradle", "build.gradle.kts":
		return true

	// Python.
	// TODO(dgageot): When the Procfile is missing, we might want to inform the user
	// that this still might be a valid python project.
	case "requirements.txt":
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "Procfile")); err == nil {
			return true
		}
	}

	return false
}
