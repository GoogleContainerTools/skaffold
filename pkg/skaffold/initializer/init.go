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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	survey "gopkg.in/AlecAivazis/survey.v1"
	yaml "gopkg.in/yaml.v2"
)

// NoBuilder allows users to specify they don't want to build
// an image we parse out from a kubernetes manifest
const NoBuilder = "None (image not built from these sources)"

// Initializer is the Init API of skaffold and responsible for generating
// skaffold configuration file.
type Initializer interface {
	// GenerateDeployConfig generates Deploy Config for skaffold configuration.
	GenerateDeployConfig() latest.DeployConfig
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
}

// InitBuilder represents a builder that can be chosen by skaffold init.
type InitBuilder interface {
	// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder
	Describe() string
	// GetArtifact creates an Artifact to be included in the generated Build Config
	GetArtifact(image string) *latest.Artifact
	// GetConfiguredImage returns the target image configured by the builder
	GetConfiguredImage() string
	// GetPath returns the path to the build file
	GetPath() string
}

// Config defines the Initializer Config for Init API of skaffold.
type Config struct {
	ComposeFile  string
	CliArtifacts []string
	SkipBuild    bool
	Force        bool
	Analyze      bool
	Opts         *config.SkaffoldOptions
}

// BuilderImagePair defines a builder and the image it builds
type BuilderImagePair struct {
	Builder   InitBuilder
	ImageName string
}

// DoInit executes the `skaffold init` flow.
func DoInit(out io.Writer, c Config) error {
	rootDir := "."

	if c.ComposeFile != "" {
		// run kompose first to generate k8s manifests, then run skaffold init
		logrus.Infof("running 'kompose convert' for file %s", c.ComposeFile)
		komposeCmd := exec.Command("kompose", "convert", "-f", c.ComposeFile)
		if err := util.RunCmd(komposeCmd); err != nil {
			return errors.Wrap(err, "running kompose")
		}
	}

	potentialConfigs, buildConfigs, err := walk(rootDir, c.Force, detectBuildFile)
	if err != nil {
		return err
	}

	k, err := kubectl.New(potentialConfigs)
	if err != nil {
		return err
	}
	images := k.GetImages()
	if c.Analyze {
		return printAnalyzeJSON(out, c.SkipBuild, buildConfigs, images)
	}

	// conditionally generate build artifacts
	var pairs []BuilderImagePair
	if !c.SkipBuild {
		if len(buildConfigs) == 0 {
			return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least Dockerfile and try again or run `skaffold init --skip-build`")
		}

		var filteredImages []string
		pairs, filteredImages = autoSelectBuilders(buildConfigs, images)

		if c.CliArtifacts != nil {
			newPairs, err := processCliArtifacts(c.CliArtifacts)
			if err != nil {
				return errors.Wrap(err, "processing cli artifacts")
			}
			pairs = append(pairs, newPairs...)
		} else {
			pairs = append(pairs, resolveBuilderImages(buildConfigs, filteredImages)...)
		}
	}

	pipeline, err := generateSkaffoldConfig(k, pairs)
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

// autoSelectBuilders takes a list of builders and images, checks if any of the builders' configured target
// images match an image in the image list, and returns a list of the matching builder/image pairs. Also
// returns the images from the original image list that didn't match any build configurations.
func autoSelectBuilders(buildConfigs []InitBuilder, images []string) ([]BuilderImagePair, []string) {
	// Auto-select builders that have a definite target image
	pairs := []BuilderImagePair{}
	filteredImages := []string{}
	for _, image := range images {
		matchingConfigIndex := -1
		for i, config := range buildConfigs {
			if image != config.GetConfiguredImage() {
				continue
			}

			// Found more than one match; can't auto-select.
			if matchingConfigIndex != -1 {
				matchingConfigIndex = -1
				break
			}
			matchingConfigIndex = i
		}

		if matchingConfigIndex != -1 {
			// Exactly one pair found; save the pair and remove from remaining build configs
			pairs = append(pairs, BuilderImagePair{ImageName: image, Builder: buildConfigs[matchingConfigIndex]})
			buildConfigs = append(buildConfigs[:matchingConfigIndex], buildConfigs[matchingConfigIndex+1:]...)
		} else {
			// No definite pair found, add to images list
			filteredImages = append(filteredImages, image)
		}
	}
	return pairs, filteredImages
}

func detectBuildFile(path string) ([]InitBuilder, error) {
	// Check for Dockerfile
	if docker.ValidateDockerfile(path) {
		results := []InitBuilder{docker.Dockerfile(path)}
		return results, nil
	}

	// TODO: Check for more builders

	return nil, nil
}

func processCliArtifacts(artifacts []string) ([]BuilderImagePair, error) {
	var pairs []BuilderImagePair
	for _, artifact := range artifacts {
		parts := strings.Split(artifact, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed artifact provided: %s", artifact)
		}

		pairs = append(pairs, BuilderImagePair{
			Builder:   docker.Dockerfile(parts[0]),
			ImageName: parts[1],
		})
	}
	return pairs, nil
}

// For each image parsed from all k8s manifests, prompt the user for the builder that builds the referenced image
func resolveBuilderImages(buildConfigs []InitBuilder, images []string) []BuilderImagePair {
	// If nothing to choose, don't bother prompting
	if len(images) == 0 || len(buildConfigs) == 0 {
		return []BuilderImagePair{}
	}

	// if we only have 1 image and 1 build config, don't bother prompting
	if len(images) == 1 && len(buildConfigs) == 1 {
		return []BuilderImagePair{{
			Builder:   buildConfigs[0],
			ImageName: images[0],
		}}
	}

	// Build map from choice string to builder config struct
	choices := make([]string, len(buildConfigs))
	choiceMap := make(map[string]InitBuilder, len(buildConfigs))
	for i, buildConfig := range buildConfigs {
		choice := buildConfig.Describe()
		choices[i] = choice
		choiceMap[choice] = buildConfig
	}

	// For each choice, use prompt string to pair builder config with k8s image
	pairs := []BuilderImagePair{}
	for {
		if len(images) == 0 {
			break
		}
		image := images[0]
		choice := promptUserForBuildConfig(image, choices)
		if choice != NoBuilder {
			pairs = append(pairs, BuilderImagePair{Builder: choiceMap[choice], ImageName: image})
			choices = util.RemoveFromSlice(choices, choice)
		}
		images = util.RemoveFromSlice(images, image)
	}
	if len(buildConfigs) > 0 {
		logrus.Warnf("unused builder configs found in repository: %v", buildConfigs)
	}
	return pairs
}

var promptUserForBuildConfig = func(image string, choices []string) string {
	var selectedBuildConfig string
	options := append(choices, NoBuilder)
	prompt := &survey.Select{
		Message:  fmt.Sprintf("Choose the builder to build image %s", image),
		Options:  options,
		PageSize: 15,
	}
	survey.AskOne(prompt, &selectedBuildConfig, nil)
	return selectedBuildConfig
}

func processBuildArtifacts(pairs []BuilderImagePair) latest.BuildConfig {
	var config latest.BuildConfig
	if len(pairs) > 0 {
		config.Artifacts = make([]*latest.Artifact, len(pairs))
		for i, pair := range pairs {
			config.Artifacts[i] = pair.Builder.GetArtifact(pair.ImageName)
		}
	}
	return config
}

func generateSkaffoldConfig(k Initializer, buildConfigPairs []BuilderImagePair) ([]byte, error) {
	// if we're here, the user has no skaffold yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	logrus.Info("generating skaffold config")

	cfg := &latest.SkaffoldConfig{
		APIVersion: latest.Version,
		Kind:       "Config",
	}
	if err := defaults.Set(cfg); err != nil {
		return nil, errors.Wrap(err, "generating default pipeline")
	}

	cfg.Build = processBuildArtifacts(buildConfigPairs)
	cfg.Deploy = k.GenerateDeployConfig()

	pipelineStr, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling generated pipeline")
	}

	return pipelineStr, nil
}

func printAnalyzeJSON(out io.Writer, skipBuild bool, buildConfigs []InitBuilder, images []string) error {
	if !skipBuild && len(buildConfigs) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again, or run `skaffold init --skip-build`")
	}
	a := struct {
		Builders []InitBuilder `json:"builders,omitempty"`
		Images   []string      `json:"images,omitempty"`
	}{
		Builders: buildConfigs,
		Images:   images,
	}
	contents, err := json.Marshal(a)
	if err != nil {
		return errors.Wrap(err, "marshalling contents")
	}
	_, err = out.Write(contents)
	return err
}

func walk(dir string, force bool, validateBuildFile func(string) ([]InitBuilder, error)) ([]string, []InitBuilder, error) {
	var potentialConfigs []string
	var buildFiles []InitBuilder
	err := filepath.Walk(dir, func(path string, f os.FileInfo, e error) error {
		if f.IsDir() && util.IsHiddenDir(f.Name()) {
			logrus.Debugf("skip walking hidden dir %s", f.Name())
			return filepath.SkipDir
		}
		if f.IsDir() || util.IsHiddenFile(f.Name()) {
			return nil
		}
		if IsSkaffoldConfig(path) {
			if !force {
				return fmt.Errorf("pre-existing %s found", path)
			}
			logrus.Debugf("%s is a valid skaffold configuration: continuing since --force=true", path)
			return nil
		}
		if IsSupportedKubernetesFileExtension(path) {
			potentialConfigs = append(potentialConfigs, path)
			return nil
		}
		// try and parse build file
		if builderConfigs, err := validateBuildFile(path); builderConfigs != nil {
			for _, buildConfig := range builderConfigs {
				logrus.Infof("existing builder found: %s", buildConfig.Describe())
				buildFiles = append(buildFiles, buildConfig)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return potentialConfigs, buildFiles, nil
}
