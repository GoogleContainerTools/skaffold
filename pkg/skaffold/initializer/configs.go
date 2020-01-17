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
	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

// checkConfigFile checks if filePath is a skaffold config or k8s config, or builder config. Detected k8s configs are added to potentialConfigs.
// Returns true if filePath is a config file, and false if not.
func checkConfigFile(filePath string, force bool, potentialConfigs *[]string) (bool, error) {
	if IsSkaffoldConfig(filePath) {
		if !force {
			return true, fmt.Errorf("pre-existing %s found (you may continue with --force)", filePath)
		}
		logrus.Debugf("%s is a valid skaffold configuration: continuing since --force=true", filePath)
		return true, nil
	}

	if kubectl.IsKubernetesManifest(filePath) {
		*potentialConfigs = append(*potentialConfigs, filePath)
		return true, nil
	}

	return false, nil
}

func generateSkaffoldConfig(k Initializer, buildConfigPairs []builderImagePair) ([]byte, error) {
	// if we're here, the user has no skaffold yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	logrus.Info("generating skaffold config")

	name, err := suggestConfigName()
	if err != nil {
		warnings.Printf("Couldn't generate default config name: %s", err.Error())
	}

	return yaml.Marshal(&latest.SkaffoldConfig{
		APIVersion: latest.Version,
		Kind:       "Config",
		Metadata: latest.Metadata{
			Name: name,
		},
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: artifacts(buildConfigPairs),
			},
			Deploy: k.GenerateDeployConfig(),
		},
	})
}

func suggestConfigName() (string, error) {
	cwd, err := os.Getwd()
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
