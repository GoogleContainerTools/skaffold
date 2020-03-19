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

package docker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moby/buildkit/frontend/dockerfile/command"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	Validate = validate
)

// Name is the name of the Docker builder
var Name = "Docker"

// ArtifactConfig holds information about a Docker build based project
type ArtifactConfig struct {
	File string `json:"path"`
}

// Name returns the name of the builder, "Docker"
func (c ArtifactConfig) Name() string {
	return Name
}

// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
func (c ArtifactConfig) Describe() string {
	return fmt.Sprintf("%s (%s)", c.Name(), c.File)
}

// ArtifactType returns the type of the artifact to be built.
func (c ArtifactConfig) ArtifactType() latest.ArtifactType {
	dockerfile := filepath.Base(c.File)
	if dockerfile == constants.DefaultDockerfilePath {
		return latest.ArtifactType{}
	}

	return latest.ArtifactType{
		DockerArtifact: &latest.DockerArtifact{
			DockerfilePath: dockerfile,
		},
	}
}

// ConfiguredImage returns the target image configured by the builder, or an empty string if no image is configured
func (c ArtifactConfig) ConfiguredImage() string {
	// Target image is not configured in dockerfiles
	return ""
}

// Path returns the path to the dockerfile
func (c ArtifactConfig) Path() string {
	return c.File
}

// validateConfig makes sure the given Dockerfile is existing and valid.
func validate(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		logrus.Warnf("opening file %s: %s", path, err.Error())
		return false
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil || res == nil || len(res.AST.Children) == 0 {
		return false
	}

	// validate each node contains valid dockerfile directive
	for _, child := range res.AST.Children {
		_, ok := command.Commands[child.Value]
		if !ok {
			return false
		}
	}

	return true
}
