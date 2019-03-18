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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
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

// NoDockerfile allows users to specify they don't want to build
// an image we parse out from a kubernetes manifest
const NoDockerfile = "None (image not built from these sources)"

// Initializer is the Init API of skaffold and responsible for generating
// skaffold configuration file.
type Initializer interface {
	// GenerateDeployConfig generates Deploy Config for skaffold configuration.
	GenerateDeployConfig() latest.DeployConfig
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
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

	var potentialConfigs, dockerfiles []string

	err := filepath.Walk(rootDir, func(path string, f os.FileInfo, e error) error {
		if f.IsDir() && util.IsHiddenDir(f.Name()) {
			logrus.Debugf("skip walking hidden dir %s", f.Name())
			return filepath.SkipDir
		}
		if f.IsDir() || util.IsHiddenFile(f.Name()) {
			return nil
		}
		if IsSkaffoldConfig(path) {
			if !c.Force {
				return fmt.Errorf("pre-existing %s found", path)
			}
			logrus.Debugf("%s is a valid skaffold configuration: continuing since --force=true", path)
			return nil
		}
		if IsSupportedKubernetesFileExtension(path) {
			potentialConfigs = append(potentialConfigs, path)
			return nil
		}
		// try and parse dockerfile
		if docker.ValidateDockerfile(path) {
			logrus.Infof("existing dockerfile found: %s", path)
			dockerfiles = append(dockerfiles, path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	k, err := kubectl.New(potentialConfigs)
	if err != nil {
		return err
	}
	images := k.GetImages()
	if c.Analyze {
		return printAnalyzeJSON(out, c.SkipBuild, dockerfiles, images)
	}
	var pairs []dockerfilePair
	// conditionally generate build artifacts
	if !c.SkipBuild {
		if len(dockerfiles) == 0 {
			return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again or run `skaffold init --skip-build`")
		}

		if c.CliArtifacts != nil {
			pairs, err = processCliArtifacts(c.CliArtifacts)
			if err != nil {
				return errors.Wrap(err, "processing cli artifacts")
			}
		} else {
			pairs = resolveDockerfileImages(dockerfiles, images)
		}
	}

	pipeline, err := generateSkaffoldPipeline(k, pairs)
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

func processCliArtifacts(artifacts []string) ([]dockerfilePair, error) {
	var pairs []dockerfilePair
	for _, artifact := range artifacts {
		parts := strings.Split(artifact, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed artifact provided: %s", artifact)
		}
		pairs = append(pairs, dockerfilePair{
			Dockerfile: parts[0],
			ImageName:  parts[1],
		})
	}
	return pairs, nil
}

// For each image parsed from all k8s manifests, prompt the user for
// the dockerfile that builds the referenced image
func resolveDockerfileImages(dockerfiles []string, images []string) []dockerfilePair {
	// if we only have 1 image and 1 dockerfile, don't bother prompting
	if len(images) == 1 && len(dockerfiles) == 1 {
		return []dockerfilePair{{
			Dockerfile: dockerfiles[0],
			ImageName:  images[0],
		}}
	}
	pairs := []dockerfilePair{}
	for {
		if len(images) == 0 {
			break
		}
		image := images[0]
		pair := promptUserForDockerfile(image, dockerfiles)
		if pair.Dockerfile != NoDockerfile {
			pairs = append(pairs, pair)
			dockerfiles = util.RemoveFromSlice(dockerfiles, pair.Dockerfile)
		}
		images = util.RemoveFromSlice(images, pair.ImageName)
	}
	if len(dockerfiles) > 0 {
		logrus.Warnf("unused dockerfiles found in repository: %v", dockerfiles)
	}
	return pairs
}

func promptUserForDockerfile(image string, dockerfiles []string) dockerfilePair {
	var selectedDockerfile string
	options := append(dockerfiles, NoDockerfile)
	prompt := &survey.Select{
		Message:  fmt.Sprintf("Choose the dockerfile to build image %s", image),
		Options:  options,
		PageSize: 15,
	}
	survey.AskOne(prompt, &selectedDockerfile, nil)
	return dockerfilePair{
		Dockerfile: selectedDockerfile,
		ImageName:  image,
	}
}

func processBuildArtifacts(pairs []dockerfilePair) latest.BuildConfig {
	var config latest.BuildConfig

	if len(pairs) > 0 {
		var artifacts []*latest.Artifact
		for _, pair := range pairs {
			workspace := filepath.Dir(pair.Dockerfile)
			dockerfilePath := filepath.Base(pair.Dockerfile)
			a := &latest.Artifact{
				ImageName: pair.ImageName,
			}
			if workspace != "." {
				a.Workspace = workspace
			}
			if dockerfilePath != constants.DefaultDockerfilePath {
				a.ArtifactType = latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: dockerfilePath,
					},
				}
			}
			artifacts = append(artifacts, a)
		}
		config.Artifacts = artifacts
	}
	return config
}

func generateSkaffoldPipeline(k Initializer, dockerfilePairs []dockerfilePair) ([]byte, error) {
	// if we're here, the user has no skaffold yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	logrus.Info("generating skaffold config")

	pipeline := &latest.SkaffoldPipeline{
		APIVersion: latest.Version,
		Kind:       "Config",
	}
	if err := defaults.Set(pipeline); err != nil {
		return nil, errors.Wrap(err, "generating default pipeline")
	}

	pipeline.Build = processBuildArtifacts(dockerfilePairs)
	pipeline.Deploy = k.GenerateDeployConfig()

	pipelineStr, err := yaml.Marshal(pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling generated pipeline")
	}

	return pipelineStr, nil
}

func printAnalyzeJSON(out io.Writer, skipBuild bool, dockerfiles, images []string) error {
	if !skipBuild && len(dockerfiles) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to build images with skaffold; please provide at least one Dockerfile and try again or run `skaffold init --skip-build`")
	}
	a := struct {
		Dockerfiles []string `json:"dockerfiles,omitempty"`
		Images      []string `json:"images,omitempty"`
	}{
		Dockerfiles: dockerfiles,
		Images:      images,
	}
	contents, err := json.Marshal(a)
	if err != nil {
		return errors.Wrap(err, "marshalling contents")
	}
	_, err = out.Write(contents)
	return err
}

type dockerfilePair struct {
	Dockerfile string
	ImageName  string
}
