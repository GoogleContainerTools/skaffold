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

package misc

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

const (
	Docker    = "docker"
	Kaniko    = "kaniko"
	Bazel     = "bazel"
	Jib       = "jib"
	Custom    = "custom"
	Buildpack = "buildpack"
)

// ArtifactType returns a string representing the type found in an artifact. Used for error messages.
// (this would normally be implemented as a String() method on the type, but types are versioned)
func ArtifactType(a *latest.Artifact) string {
	switch {
	case a.DockerArtifact != nil:
		return Docker
	case a.KanikoArtifact != nil:
		return Kaniko
	case a.BazelArtifact != nil:
		return Bazel
	case a.JibArtifact != nil:
		return Jib
	case a.CustomArtifact != nil:
		return Custom
	case a.BuildpackArtifact != nil:
		return Buildpack
	default:
		return ""
	}
}

// FormatArtifact returns a string representation of an artifact for usage in error messages
func FormatArtifact(a *latest.Artifact) string {
	buf, err := yaml.Marshal(a)
	if err != nil {
		return fmt.Sprintf("%+v", a)
	}
	return strings.TrimSpace(string(buf))
}
