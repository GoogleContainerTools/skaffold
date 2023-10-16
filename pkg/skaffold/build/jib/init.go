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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// For testing
var (
	Validate = validate
)

// ArtifactConfig holds information about a Jib project
type ArtifactConfig struct {
	BuilderName string `json:"-"`
	Image       string `json:"image,omitempty"`
	File        string `json:"path,omitempty"`
	Project     string `json:"project,omitempty"`
}

// Name returns the name of the builder
func (c ArtifactConfig) Name() string {
	return c.BuilderName
}

// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
func (c ArtifactConfig) Describe() string {
	if c.Project != "" {
		return fmt.Sprintf("%s (%s, %s)", c.BuilderName, c.Project, c.File)
	}
	return fmt.Sprintf("%s (%s)", c.BuilderName, c.File)
}

// ArtifactType returns the type of the artifact to be built.
func (c ArtifactConfig) ArtifactType(_ string) latest.ArtifactType {
	return latest.ArtifactType{
		JibArtifact: &latest.JibArtifact{
			Project: c.Project,
		},
	}
}

// ConfiguredImage returns the target image configured by the builder, or empty string if no image is configured
func (c ArtifactConfig) ConfiguredImage() string {
	return c.Image
}

// Path returns the path to the build definition
func (c ArtifactConfig) Path() string {
	return c.File
}

// BuilderConfig contains information about inferred Jib configurations
type jibJSON struct {
	Image   string `json:"image"`
	Project string `json:"project"`
}

// validate checks if a file is a valid Jib configuration. Returns the list of Config objects corresponding to each Jib project built by the file, or nil if Jib is not configured.
func validate(ctx context.Context, path string, enableGradleAnalysis bool) []ArtifactConfig {
	if !JVMFound(ctx) {
		log.Entry(context.TODO()).Debugf("Skipping Jib for init for %q: no functioning Java VM", path)
		return nil
	}
	// Determine whether maven or gradle
	var builderType PluginType
	var executable, wrapper, taskName, searchString, consoleFlag string
	switch {
	case isPomFile(path):
		builderType = JibMaven
		executable = "mvn"
		wrapper = "mvnw"
		searchString = "jib-maven-plugin"
		taskName = "jib:_skaffold-init"
		consoleFlag = "--batch-mode"

	case enableGradleAnalysis && (strings.HasSuffix(path, "build.gradle") || strings.HasSuffix(path, "build.gradle.kts")):
		builderType = JibGradle
		executable = "gradle"
		wrapper = "gradlew"
		searchString = "com.google.cloud.tools.jib"
		taskName = "_jibSkaffoldInit"
		consoleFlag = "--console=plain"
	default:
		return nil
	}

	// Search for indication of Jib in build file before proceeding
	if content, err := os.ReadFile(path); err != nil || !strings.Contains(string(content), searchString) {
		return nil
	}

	// Run Jib's skaffold init task/goal to check if Jib is configured
	if wrapperExecutable, err := util.AbsFile(filepath.Dir(path), wrapper); err == nil {
		executable = wrapperExecutable
	}
	cmd := exec.Command(executable, taskName, "-q", consoleFlag)
	cmd.Dir = filepath.Dir(path)
	stdout, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return nil
	}

	// Parse Jib output. Multiple JSON strings may be printed for multi-project/multi-module setups.
	matches := regexp.MustCompile(`BEGIN JIB JSON\r?\n({.*})`).FindAllSubmatch(stdout, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []ArtifactConfig
	for _, match := range matches {
		// Escape windows path separators
		line := bytes.ReplaceAll(match[1], []byte(`\`), []byte(`\\`))
		parsedJSON := jibJSON{}
		if err := json.Unmarshal(line, &parsedJSON); err != nil {
			log.Entry(context.TODO()).Warnf("failed to parse jib json: %s", err.Error())
			return nil
		}

		results = append(results, ArtifactConfig{
			BuilderName: PluginName(builderType),
			Image:       parsedJSON.Image,
			File:        path,
			Project:     parsedJSON.Project,
		})
	}
	return results
}

// checks that the file is a maven pom file, and returns the file extension
func isPomFile(path string) bool {
	filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return filename == "pom"
}
