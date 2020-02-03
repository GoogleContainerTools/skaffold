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

package initializer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

var (
	// for testing
	getWd = os.Getwd
)

type skaffoldConfigAnalyzer struct {
	directoryAnalyzer
	force        bool
	analyzeMode  bool
	targetConfig string
}

func (a *skaffoldConfigAnalyzer) analyzeFile(filePath string) error {
	if !isSkaffoldConfig(filePath) || a.force || a.analyzeMode {
		return nil
	}
	sameFiles, err := sameFiles(filePath, a.targetConfig)
	if err != nil {
		return fmt.Errorf("failed to analyze file %s: %s", filePath, err)
	}
	if !sameFiles {
		return nil
	}
	return fmt.Errorf("pre-existing %s found (you may continue with --force)", filePath)
}

func sameFiles(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return absA == absB, nil
}

func generateSkaffoldConfig(k deploymentInitializer, buildConfigPairs []builderImagePair) *latest.SkaffoldConfig {
	// if we're here, the user has no skaffold yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	logrus.Info("generating skaffold config")

	name, err := suggestConfigName()
	if err != nil {
		warnings.Printf("Couldn't generate default config name: %s", err.Error())
	}

	return &latest.SkaffoldConfig{
		APIVersion: latest.Version,
		Kind:       "Config",
		Metadata: latest.Metadata{
			Name: name,
		},
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: artifacts(buildConfigPairs),
			},
			Deploy: k.deployConfig(),
		},
	}
}

func suggestConfigName() (string, error) {
	cwd, err := getWd()
	if err != nil {
		return "", err
	}

	base := filepath.Base(cwd)

	// give up for edge cases
	if base == "." || base == string(filepath.Separator) {
		return "", nil
	}

	return canonicalizeName(base), nil
}

// canonicalizeName converts a given string to a valid k8s name string.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names for details
func canonicalizeName(name string) string {
	forbidden := regexp.MustCompile(`[^-.a-z]+`)
	canonicalized := forbidden.ReplaceAllString(strings.ToLower(name), "-")
	if len(canonicalized) <= 253 {
		return canonicalized
	}
	return canonicalized[:253]
}

func artifacts(pairs []builderImagePair) []*latest.Artifact {
	var artifacts []*latest.Artifact

	for _, pair := range pairs {
		artifact := &latest.Artifact{
			ImageName: pair.ImageName,
		}

		workspace := filepath.Dir(pair.Builder.Path())
		if workspace != "." {
			artifact.Workspace = workspace
		}

		pair.Builder.UpdateArtifact(artifact)

		artifacts = append(artifacts, artifact)
	}

	return artifacts
}

// isSkaffoldConfig is for determining if a file is skaffold config file.
func isSkaffoldConfig(file string) bool {
	if config, err := schema.ParseConfig(file, false); err == nil && config != nil {
		return true
	}
	return false
}
