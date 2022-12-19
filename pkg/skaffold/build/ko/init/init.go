/*
Copyright 2022 The Skaffold Authors

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

package init

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// ArtifactConfig holds information about a Ko project
type ArtifactConfig struct {
	File string `json:"path,omitempty"`
}

var _ build.InitBuilder = &ArtifactConfig{}

// Validate checks if the file is a Go module file.
func Validate(path string) bool {
	return filepath.Base(path) == "go.mod"
}

// Name returns the name of the builder.
func (c ArtifactConfig) Name() string {
	return "Ko"
}

// Describe returns the initBuilder's string representation.
// This representation is used when prompting the user to choose a builder.
func (c ArtifactConfig) Describe() string {
	return fmt.Sprintf("%s (%s)", c.Name(), c.File)
}

// ArtifactType returns a definition of the artifact to be built.
func (c ArtifactConfig) ArtifactType(_ string) latest.ArtifactType {
	return latest.ArtifactType{
		KoArtifact: &latest.KoArtifact{
			Dependencies: &latest.KoDependencies{
				Paths: []string{"**/*.go", "go.*"},
			},
		},
	}
}

// ConfiguredImage returns the target image configured by the builder, or empty string if no image is configured.
func (c ArtifactConfig) ConfiguredImage() string {
	// Target image is not configured in Ko.
	return ""
}

// Path returns the path to the build definition.
func (c ArtifactConfig) Path() string {
	return c.File
}
