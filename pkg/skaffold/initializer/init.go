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
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NoBuilder allows users to specify they don't want to build
// an image we parse out from a Kubernetes manifest
const NoBuilder = "None (image not built from these sources)"

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
	CliKubectlManifests []string
	SkipBuild           bool
	SkipDeploy          bool
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

	a := newAnalysis(c)

	if err := a.analyze(rootDir); err != nil {
		return err
	}

	var deployInitializer deploymentInitializer
	switch {
	case c.SkipDeploy:
		deployInitializer = &emptyDeployInit{}
	case len(c.CliKubectlManifests) > 0:
		deployInitializer = &cliDeployInit{c.CliKubectlManifests}
	default:
		k, err := newKubectlInitializer(a.kubectlAnalyzer.kubernetesManifests)
		if err != nil {
			return err
		}
		deployInitializer = k
	}

	// Determine which builders/images require prompting
	pairs, unresolvedBuilderConfigs, unresolvedImages :=
		matchBuildersToImages(
			a.builderAnalyzer.foundBuilders,
			stripTags(deployInitializer.GetImages()))

	if c.Analyze {
		// TODO: Remove backwards compatibility block
		if !c.EnableJibInit && !c.EnableBuildpackInit {
			return printAnalyzeJSONNoJib(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
		}

		return printAnalyzeJSON(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
	}
	if !c.SkipBuild {
		if len(a.builderAnalyzer.foundBuilders) == 0 && c.CliArtifacts == nil {
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

	pipeline, err := yaml.Marshal(generateSkaffoldConfig(deployInitializer, pairs))
	if err != nil {
		return err
	}
	if c.Opts.ConfigurationFile == "-" {
		out.Write(pipeline)
		return nil
	}

	if !c.Force {
		if done, err := promptWritingConfig(out, pipeline, c.Opts.ConfigurationFile); done {
			return err
		}
	}

	if err := ioutil.WriteFile(c.Opts.ConfigurationFile, pipeline, 0644); err != nil {
		return errors.Wrap(err, "writing config to file")
	}

	fmt.Fprintf(out, "Configuration %s was written\n", c.Opts.ConfigurationFile)
	tips.PrintForInit(out, c.Opts)

	return nil
}
