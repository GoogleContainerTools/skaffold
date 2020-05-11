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

package generatepipeline

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	yamlv2 "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func CreateSkaffoldProfile(out io.Writer, runCtx *runcontext.RunContext, configFile *ConfigFile) error {
	reader := bufio.NewReader(os.Stdin)

	// Check for existing oncluster profile, if none exists then prompt to create one
	color.Default.Fprintf(out, "Checking for oncluster skaffold profile in %s...\n", configFile.Path)
	for _, profile := range configFile.Config.Profiles {
		if profile.Name == "oncluster" {
			color.Default.Fprintln(out, "profile \"oncluster\" found")
			configFile.Profile = &profile
			return nil
		}
	}

confirmLoop:
	for {
		color.Default.Fprintf(out, "No profile \"oncluster\" found. Create one? [y/n]: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading user confirmation: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y", "yes":
			break confirmLoop
		case "n", "no":
			return nil
		}
	}

	color.Default.Fprintln(out, "Creating skaffold profile \"oncluster\"...")
	profile, err := generateProfile(out, runCtx.Opts.Namespace, configFile.Config)
	if err != nil {
		return fmt.Errorf("generating profile \"oncluster\": %w", err)
	}

	bProfile, err := yamlv2.Marshal([]*latest.Profile{profile})
	if err != nil {
		return fmt.Errorf("marshaling new profile: %w", err)
	}

	fileContents, err := ioutil.ReadFile(configFile.Path)
	if err != nil {
		return fmt.Errorf("reading file contents: %w", err)
	}
	fileStrings := strings.Split(strings.TrimSpace(string(fileContents)), "\n")

	var profilePos int
	if len(configFile.Config.Profiles) == 0 {
		// Create new profiles section
		fileStrings = append(fileStrings, "profiles:")
		profilePos = len(fileStrings)
	} else {
		for i, line := range fileStrings {
			if line == "profiles:" {
				profilePos = i + 1
			}
		}
	}

	fileStrings = append(fileStrings, "")
	copy(fileStrings[profilePos+1:], fileStrings[profilePos:])
	fileStrings[profilePos] = strings.TrimSpace(string(bProfile))

	fileContents = []byte((strings.Join(fileStrings, "\n")))

	if err := ioutil.WriteFile(configFile.Path, fileContents, 0644); err != nil {
		return fmt.Errorf("writing profile to skaffold config: %w", err)
	}

	configFile.Profile = profile
	return nil
}

func generateProfile(out io.Writer, namespace string, config *latest.SkaffoldConfig) (*latest.Profile, error) {
	if len(config.Build.Artifacts) == 0 {
		return nil, errors.New("no Artifacts to add to profile")
	}

	profile := &latest.Profile{
		Name: "oncluster",
		Pipeline: latest.Pipeline{
			Build:  config.Pipeline.Build,
			Deploy: latest.DeployConfig{},
		},
	}

	// Add kaniko build config for artifacts
	addKaniko := false
	for _, artifact := range profile.Build.Artifacts {
		artifact.ImageName = fmt.Sprintf("%s-pipeline", artifact.ImageName)
		if artifact.DockerArtifact != nil {
			color.Default.Fprintf(out, "Cannot use Docker to build %s on cluster. Adding config for building with Kaniko.\n", artifact.ImageName)
			artifact.DockerArtifact = nil
			artifact.KanikoArtifact = &latest.KanikoArtifact{}
			addKaniko = true
		}
	}
	// Add kaniko config to build config if needed
	if addKaniko {
		profile.Build.Cluster = &latest.ClusterDetails{
			PullSecretName: "kaniko-secret",
		}
		profile.Build.LocalBuild = nil
	}
	if namespace != "" {
		if profile.Build.Cluster == nil {
			profile.Build.Cluster = &latest.ClusterDetails{}
		}
		profile.Build.Cluster.Namespace = namespace
	}

	return profile, nil
}
