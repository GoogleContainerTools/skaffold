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

package jib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	// JibGradle the name of the Jib Gradle Plugin
	JibGradle = "Jib Gradle Plugin"
	// JibMaven the name of the Jib Maven Plugin
	JibMaven = "Jib Maven Plugin"
)

// Config holds information about a Jib project
type Config struct {
	Name    string `json:"name,omitempty"`
	Image   string `json:"image,omitempty"`
	Path    string `json:"path,omitempty"`
	Project string `json:"project,omitempty"`
}

// GetPrompt returns the initBuilder's string representation, used when prompting the user to choose a builder.
func (j Config) GetPrompt() string {
	if j.Project != "" {
		return fmt.Sprintf("%s (%s, %s)", j.Name, j.Project, j.Path)
	}
	return fmt.Sprintf("%s (%s)", j.Name, j.Path)
}

// GetArtifact returns the Artifact used to generate the Build Config.
func (j Config) GetArtifact(manifestImage string) *latest.Artifact {
	path := string(j.Path)
	workspace := filepath.Dir(path)
	a := &latest.Artifact{ImageName: manifestImage}
	if workspace != "." {
		a.Workspace = workspace
	}

	if j.Name == JibMaven {
		jibMaven := &latest.JibMavenArtifact{}
		if j.Project != "" {
			jibMaven.Module = j.Project
		}
		if j.Image == "" {
			jibMaven.Flags = []string{"-Dimage=" + manifestImage}
		}
		a.ArtifactType = latest.ArtifactType{JibMavenArtifact: jibMaven}

	} else if j.Name == JibGradle {
		jibGradle := &latest.JibGradleArtifact{}
		if j.Project != "" {
			jibGradle.Project = j.Project
		}
		if j.Image == "" {
			jibGradle.Flags = []string{"-Dimage=" + manifestImage}
		}
		a.ArtifactType = latest.ArtifactType{JibGradleArtifact: jibGradle}
	}

	return a
}

// GetConfiguredImage returns the target image configured by the builder
func (j Config) GetConfiguredImage() string {
	return j.Image
}

// GetPath returns the path to the build definition
func (j Config) GetPath() string {
	return j.Path
}

// BuilderConfig contains information about inferred Jib configurations
type jibJSON struct {
	Image   string `json:"image"`
	Project string `json:"project"`
}

// ValidateJibConfig checks if a file is a valid Jib configuration. Returns the list of Config objects corresponding to each Jib project built by the file, or nil if Jib is not configured.
var ValidateJibConfig = func(path string) []Config {
	// Determine whether maven or gradle
	var builderType, executable, wrapper, taskName string
	if strings.HasSuffix(path, "pom.xml") {
		builderType = JibMaven
		executable = "mvn"
		wrapper = "mvnw"
		taskName = "jib:_skaffold-init"
	} else if strings.HasSuffix(path, "build.gradle") {
		builderType = JibGradle
		executable = "gradle"
		wrapper = "gradlew"
		taskName = "_jibSkaffoldInit"
	} else {
		return nil
	}

	// Run Jib's skaffold init task/goal to check if Jib is configured
	if wrapperExecutable, err := util.AbsFile(filepath.Dir(path), wrapper); err == nil {
		executable = wrapperExecutable
	}
	cmd := exec.Command(executable, taskName, "-q")
	cmd.Dir = filepath.Dir(path)
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil
	}

	// Parse Jib output. Multiple JSON strings may be printed for multi-project/multi-module setups.
	matches := regexp.MustCompile(`BEGIN JIB JSON\r?\n({.*})`).FindAllSubmatch(stdout, -1)
	if len(matches) == 0 {
		return nil
	}

	results := make([]Config, len(matches))
	for i, match := range matches {
		line := bytes.Replace(match[1], []byte(`\`), []byte(`\\`), -1)
		parsedJSON := jibJSON{}
		if err := json.Unmarshal(line, &parsedJSON); err != nil {
			return nil
		}

		results[i] = Config{Name: builderType, Image: parsedJSON.Image, Path: path, Project: parsedJSON.Project}
	}
	return results
}
