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

package initializer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

// NoBuilder allows users to specify they don't want to build
// an image we parse out from a Kubernetes manifest
const NoBuilder = "None (image not built from these sources)"

// DeploymentInitializer detects a deployment type and is able to extract image names from it
type DeploymentInitializer interface {
	// GenerateDeployConfig generates Deploy Config for skaffold configuration.
	GenerateDeployConfig() latest.DeployConfig
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
}

// InitBuilder represents a builder that can be chosen by skaffold init.
type InitBuilder interface {
	// Name returns the name of the builder
	Name() string
	// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
	// Must be unique between artifacts.
	Describe() string
	// UpdateArtifact updates the Artifact to be included in the generated Build Config
	UpdateArtifact(*latest.Artifact)
	// ConfiguredImage returns the target image configured by the builder, or an empty string if no image is configured.
	// This should be a cheap operation.
	ConfiguredImage() string
	// Path returns the path to the build file
	Path() string
}

// Config contains all the parameters for the initializer package
type Config struct {
	ComposeFile         string
	CliArtifacts        []string
	SkipBuild           bool
	Force               bool
	Analyze             bool
	EnableJibInit       bool // TODO: Remove this parameter
	EnableBuildpackInit bool
	Opts                config.SkaffoldOptions
}

// builderImagePair defines a builder and the image it builds
type builderImagePair struct {
	Builder   InitBuilder
	ImageName string
}

// DoInit executes the `skaffold init` flow.
func DoInit(ctx context.Context, out io.Writer, c Config) error {
	rootDir := "."

	if c.ComposeFile != "" {
		if err := runKompose(ctx, c.ComposeFile); err != nil {
			return err
		}
	}

	a := &analysis{
		kubectlAnalyzer: &KubectlAnalyzer{},
		builderAnalyzer: &BuilderAnalyzer{
			findBuilders:        !c.SkipBuild,
			enableJibInit:       c.EnableJibInit,
			enableBuildpackInit: c.EnableBuildpackInit,
		},
		skaffoldAnalyzer: &SkaffoldConfigAnalyzer{
			force: c.Force,
		},
	}

	if err := a.walk(rootDir); err != nil {
		return err
	}

	k, err := kubectl.New(a.kubectlAnalyzer.kubernetesManifests)
	if err != nil {
		return err
	}

	// Remove tags from image names
	var images []string
	for _, image := range k.GetImages() {
		parsed, err := docker.ParseReference(image)
		if err != nil {
			// It's possible that it's a templatized name that can't be parsed as is.
			warnings.Printf("Couldn't parse image [%s]: %s", image, err.Error())
			continue
		}
		if parsed.Digest != "" {
			warnings.Printf("Ignoring image referenced by digest: [%s]", image)
			continue
		}

		images = append(images, parsed.BaseName)
	}

	// Determine which builders/images require prompting
	pairs, unresolvedBuilderConfigs, unresolvedImages := autoSelectBuilders(a.builderAnalyzer.foundBuilders, images)

	if c.Analyze {
		// TODO: Remove backwards compatibility block
		if !c.EnableJibInit && !c.EnableBuildpackInit {
			return printAnalyzeJSONNoJib(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
		}

		return printAnalyzeJSON(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
	}

	// conditionally generate build artifacts
	if !c.SkipBuild {
		if len(a.builderAnalyzer.foundBuilders) == 0 {
			return errors.New("one or more valid builder configuration (Dockerfile or Jib configuration) must be present to build images with skaffold; please provide at least one build config and try again or run `skaffold init --skip-build`")
		}

		if c.CliArtifacts != nil {
			newPairs, err := processCliArtifacts(c.CliArtifacts)
			if err != nil {
				return errors.Wrap(err, "processing cli artifacts")
			}
			pairs = append(pairs, newPairs...)
		} else {
			resolved, err := resolveBuilderImages(unresolvedBuilderConfigs, unresolvedImages, c.Force)
			if err != nil {
				return err
			}
			pairs = append(pairs, resolved...)
		}
	}

	pipeline, err := yaml.Marshal(generateSkaffoldConfig(k, pairs))
	if err != nil {
		return err
	}
	if c.Opts.ConfigurationFile == "-" {
		out.Write(pipeline)
		return nil
	}

	if !c.Force {
		fmt.Fprintln(out, string(pipeline))

		reader := bufio.NewReader(os.Stdin)
	confirmLoop:
		for {
			fmt.Fprintf(out, "Do you want to write this configuration to %s? [y/n]: ", c.Opts.ConfigurationFile)

			response, err := reader.ReadString('\n')
			if err != nil {
				return errors.Wrap(err, "reading user confirmation")
			}

			response = strings.ToLower(strings.TrimSpace(response))
			switch response {
			case "y", "yes":
				break confirmLoop
			case "n", "no":
				return nil
			}
		}
	}

	if err := ioutil.WriteFile(c.Opts.ConfigurationFile, pipeline, 0644); err != nil {
		return errors.Wrap(err, "writing config to file")
	}

	fmt.Fprintf(out, "Configuration %s was written\n", c.Opts.ConfigurationFile)
	tips.PrintForInit(out, c.Opts)

	return nil
}
