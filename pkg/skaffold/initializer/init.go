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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

// For testing
var (
	promptUserForBuildConfigFunc = promptUserForBuildConfig
)

// NoBuilder allows users to specify they don't want to build
// an image we parse out from a Kubernetes manifest
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

// Config defines the Initializer Config for Init API of skaffold.
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

type set map[string]interface{}

func (s set) add(value string) {
	s[value] = value
}

func (s set) values() (values []string) {
	for val := range s {
		values = append(values, val)
	}
	sort.Strings(values)
	return values
}

// DoInit executes the `skaffold init` flow.
func DoInit(ctx context.Context, out io.Writer, c Config) error {
	rootDir := "."

	if c.ComposeFile != "" {
		if err := runKompose(ctx, c.ComposeFile); err != nil {
			return err
		}
	}

	potentialConfigs, builderConfigs, err := walk(rootDir, c.Force, c.EnableJibInit, c.EnableBuildpackInit)
	if err != nil {
		return err
	}

	k, err := kubectl.New(potentialConfigs)
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
	pairs, unresolvedBuilderConfigs, unresolvedImages := autoSelectBuilders(builderConfigs, images)

	if c.Analyze {
		// TODO: Remove backwards compatibility block
		if !c.EnableJibInit && !c.EnableBuildpackInit {
			return printAnalyzeJSONNoJib(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
		}

		return printAnalyzeJSON(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
	}

	// conditionally generate build artifacts
	if !c.SkipBuild {
		if len(builderConfigs) == 0 {
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

// runKompose runs the `kompose` CLI before running skaffold init
func runKompose(ctx context.Context, composeFile string) error {
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return err
	}

	logrus.Infof("running 'kompose convert' for file %s", composeFile)
	komposeCmd := exec.CommandContext(ctx, "kompose", "convert", "-f", composeFile)
	_, err := util.RunCmdOut(komposeCmd)
	return err
}

// autoSelectBuilders takes a list of builders and images, checks if any of the builders' configured target
// images match an image in the image list, and returns a list of the matching builder/image pairs. Also
// separately returns the builder configs and images that didn't have any matches.
func autoSelectBuilders(builderConfigs []InitBuilder, images []string) ([]builderImagePair, []InitBuilder, []string) {
	var pairs []builderImagePair
	var unresolvedImages = make(set)
	for _, image := range images {
		matchingConfigIndex := -1
		for i, config := range builderConfigs {
			if image != config.ConfiguredImage() {
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
			pairs = append(pairs, builderImagePair{ImageName: image, Builder: builderConfigs[matchingConfigIndex]})
			builderConfigs = append(builderConfigs[:matchingConfigIndex], builderConfigs[matchingConfigIndex+1:]...)
		} else {
			// No definite pair found, add to images list
			unresolvedImages.add(image)
		}
	}
	return pairs, builderConfigs, unresolvedImages.values()
}

// detectBuilders checks if a path is a builder config, and if it is, returns the InitBuilders representing the
// configs. Also returns a boolean marking search completion for subdirectories (true = subdirectories should
// continue to be searched, false = subdirectories should not be searched for more builders)
func detectBuilders(enableJibInit, enableBuildpackInit bool, path string) ([]InitBuilder, bool) {
	// TODO: Remove backwards compatibility if statement (not entire block)
	if enableJibInit {
		// Check for jib
		if builders := jib.Validate(path); builders != nil {
			results := make([]InitBuilder, len(builders))
			for i := range builders {
				results[i] = builders[i]
			}
			return results, false
		}
	}

	// Check for Dockerfile
	base := filepath.Base(path)
	if strings.Contains(strings.ToLower(base), "dockerfile") {
		if docker.Validate(path) {
			results := []InitBuilder{docker.ArtifactConfig{File: path}}
			return results, true
		}
	}

	// TODO: Remove backwards compatibility if statement (not entire block)
	if enableBuildpackInit {
		// Check for buildpacks
		if buildpacks.Validate(path) {
			results := []InitBuilder{buildpacks.ArtifactConfig{File: path}}
			return results, true
		}
	}

	// TODO: Check for more builders

	return nil, true
}

func processCliArtifacts(artifacts []string) ([]builderImagePair, error) {
	var pairs []builderImagePair
	for _, artifact := range artifacts {
		// Parses JSON in the form of: {"builder":"Name of Builder","payload":{...},"image":"image.name"}.
		// The builder field is parsed first to determine the builder type, and the payload is parsed
		// afterwards once the type is determined.
		a := struct {
			Name  string `json:"builder"`
			Image string `json:"image"`
		}{}
		if err := json.Unmarshal([]byte(artifact), &a); err != nil {
			// Not JSON, use backwards compatible method
			parts := strings.Split(artifact, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed artifact provided: %s", artifact)
			}
			pairs = append(pairs, builderImagePair{
				Builder:   docker.ArtifactConfig{File: parts[0]},
				ImageName: parts[1],
			})
			continue
		}

		// Use builder type to parse payload
		switch a.Name {
		case docker.Name:
			parsed := struct {
				Payload docker.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := builderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		// FIXME: shouldn't use a human-readable name?
		case jib.PluginName(jib.JibGradle), jib.PluginName(jib.JibMaven):
			parsed := struct {
				Payload jib.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			parsed.Payload.BuilderName = a.Name
			pair := builderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		case buildpacks.Name:
			parsed := struct {
				Payload buildpacks.ArtifactConfig `json:"payload"`
			}{}
			if err := json.Unmarshal([]byte(artifact), &parsed); err != nil {
				return nil, err
			}
			pair := builderImagePair{Builder: parsed.Payload, ImageName: a.Image}
			pairs = append(pairs, pair)

		default:
			return nil, errors.New("unknown builder type in CLI artifacts")
		}
	}
	return pairs, nil
}

// For each image parsed from all k8s manifests, prompt the user for the builder that builds the referenced image
func resolveBuilderImages(builderConfigs []InitBuilder, images []string, force bool) ([]builderImagePair, error) {
	// If nothing to choose, don't bother prompting
	if len(images) == 0 || len(builderConfigs) == 0 {
		return []builderImagePair{}, nil
	}

	// if we only have 1 image and 1 build config, don't bother prompting
	if len(images) == 1 && len(builderConfigs) == 1 {
		return []builderImagePair{{
			Builder:   builderConfigs[0],
			ImageName: images[0],
		}}, nil
	}

	if force {
		return nil, errors.New("unable to automatically resolve builder/image pairs; run `skaffold init` without `--force` to manually resolve ambiguities")
	}

	// Build map from choice string to builder config struct
	choices := make([]string, len(builderConfigs))
	choiceMap := make(map[string]InitBuilder, len(builderConfigs))
	for i, buildConfig := range builderConfigs {
		choice := buildConfig.Describe()
		choices[i] = choice
		choiceMap[choice] = buildConfig
	}
	sort.Strings(choices)

	// For each choice, use prompt string to pair builder config with k8s image
	pairs := []builderImagePair{}
	for {
		if len(images) == 0 {
			break
		}

		image := images[0]
		choice, err := promptUserForBuildConfigFunc(image, choices)
		if err != nil {
			return nil, err
		}

		if choice != NoBuilder {
			pairs = append(pairs, builderImagePair{Builder: choiceMap[choice], ImageName: image})
			choices = util.RemoveFromSlice(choices, choice)
		}
		images = util.RemoveFromSlice(images, image)
	}
	if len(choices) > 0 {
		logrus.Warnf("unused builder configs found in repository: %v", choices)
	}
	return pairs, nil
}

func promptUserForBuildConfig(image string, choices []string) (string, error) {
	var selectedBuildConfig string
	options := append(choices, NoBuilder)
	prompt := &survey.Select{
		Message:  fmt.Sprintf("Choose the builder to build image %s", image),
		Options:  options,
		PageSize: 15,
	}
	err := survey.AskOne(prompt, &selectedBuildConfig, nil)
	if err != nil {
		return "", err
	}

	return selectedBuildConfig, nil
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

func printAnalyzeJSONNoJib(out io.Writer, skipBuild bool, pairs []builderImagePair, unresolvedBuilders []InitBuilder, unresolvedImages []string) error {
	if !skipBuild && len(unresolvedBuilders) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again, or run `skaffold init --skip-build`")
	}

	a := struct {
		Dockerfiles []string `json:"dockerfiles,omitempty"`
		Images      []string `json:"images,omitempty"`
	}{Images: unresolvedImages}

	for _, pair := range pairs {
		if pair.Builder.Name() == docker.Name {
			a.Dockerfiles = append(a.Dockerfiles, pair.Builder.Path())
		}
		a.Images = append(a.Images, pair.ImageName)
	}
	for _, config := range unresolvedBuilders {
		if config.Name() == docker.Name {
			a.Dockerfiles = append(a.Dockerfiles, config.Path())
		}
	}

	return printJSON(out, a)
}

// printAnalyzeJSON takes the automatically resolved builder/image pairs, the unresolved images, and the unresolved builders, and generates
// a JSON string containing builder config information,
func printAnalyzeJSON(out io.Writer, skipBuild bool, pairs []builderImagePair, unresolvedBuilders []InitBuilder, unresolvedImages []string) error {
	if !skipBuild && len(unresolvedBuilders) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again, or run `skaffold init --skip-build`")
	}

	// Build JSON output. Example schema is below:
	// {
	//     "builders":[
	//         {
	//             "name":"Docker",
	//             "payload":"path/to/Dockerfile"
	//         },
	//         {
	//             "name":"Name of Builder",
	//             "payload": { // Payload structure may vary depending on builder type
	//                 "path":"path/to/builder.config",
	//                 "targetImage":"gcr.io/project/images",
	//                 ...
	//             }
	//         },
	//     ],
	//     "images":[
	//         {"name":"gcr.io/project/images", "foundMatch":"true"}, // No need to prompt for this image since its builder was automatically resolved
	//         {"name":"another/image", "foundMatch":"false"},
	//     ],
	// }
	//
	// "builders" is the list of builder configurations, and contains a builder name and a builder-specific payload
	// "images" contains an image name and a boolean that indicates whether a builder/image pair can be automatically resolved (true) or if it requires prompting (false)
	type Builder struct {
		Name    string      `json:"name,omitempty"`
		Payload InitBuilder `json:"payload"`
	}
	type Image struct {
		Name       string `json:"name"`
		FoundMatch bool   `json:"foundMatch"`
	}
	a := struct {
		Builders []Builder `json:"builders,omitempty"`
		Images   []Image   `json:"images,omitempty"`
	}{}

	for _, pair := range pairs {
		a.Builders = append(a.Builders, Builder{Name: pair.Builder.Name(), Payload: pair.Builder})
		a.Images = append(a.Images, Image{Name: pair.ImageName, FoundMatch: true})
	}
	for _, config := range unresolvedBuilders {
		a.Builders = append(a.Builders, Builder{Name: config.Name(), Payload: config})
	}
	for _, image := range unresolvedImages {
		a.Images = append(a.Images, Image{Name: image, FoundMatch: false})
	}

	return printJSON(out, a)
}

func printJSON(out io.Writer, v interface{}) error {
	contents, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshalling contents")
	}

	_, err = out.Write(contents)
	return err
}

// walk recursively walks a directory and returns the k8s configs and builder configs that it finds
func walk(dir string, force, enableJibInit, enableBuildpackInit bool) ([]string, []InitBuilder, error) {
	var potentialConfigs []string
	var foundBuilders []InitBuilder

	var searchConfigsAndBuilders func(path string, findBuilders bool) error
	searchConfigsAndBuilders = func(path string, findBuilders bool) error {
		dirents, err := godirwalk.ReadDirents(path, nil)
		if err != nil {
			return err
		}

		var subdirectories []*godirwalk.Dirent
		searchForBuildersInSubdirectories := findBuilders
		sort.Sort(dirents)

		// Traverse files
		for _, file := range dirents {
			if util.IsHiddenFile(file.Name()) || util.IsHiddenDir(file.Name()) {
				continue
			}

			// If we found a directory, keep track of it until we've gone through all the files first
			if file.IsDir() {
				subdirectories = append(subdirectories, file)
				continue
			}

			// Check for skaffold.yaml/k8s manifest
			filePath := filepath.Join(path, file.Name())
			var foundConfig bool
			if foundConfig, err = checkConfigFile(filePath, force, &potentialConfigs); err != nil {
				return err
			}

			// Check for builder config
			if !foundConfig && findBuilders {
				builderConfigs, continueSearchingBuilders := detectBuilders(enableJibInit, enableBuildpackInit, filePath)
				foundBuilders = append(foundBuilders, builderConfigs...)
				searchForBuildersInSubdirectories = searchForBuildersInSubdirectories && continueSearchingBuilders
			}
		}

		// Recurse into subdirectories
		for _, dir := range subdirectories {
			if err = searchConfigsAndBuilders(filepath.Join(path, dir.Name()), searchForBuildersInSubdirectories); err != nil {
				return err
			}
		}

		return nil
	}

	err := searchConfigsAndBuilders(dir, true)
	if err != nil {
		return nil, nil, err
	}
	return potentialConfigs, foundBuilders, nil
}

// checkConfigFile checks if filePath is a skaffold config or k8s config, or builder config. Detected k8s configs are added to potentialConfigs.
// Returns true if filePath is a config file, and false if not.
func checkConfigFile(filePath string, force bool, potentialConfigs *[]string) (bool, error) {
	if IsSkaffoldConfig(filePath) {
		if !force {
			return true, fmt.Errorf("pre-existing %s found", filePath)
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
