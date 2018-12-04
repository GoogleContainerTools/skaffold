/*
Copyright 2018 The Skaffold Authors

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

package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// NoDockerfile allows users to specify they don't want to build
// an image we parse out from a kubernetes manifest
const NoDockerfile = "None (image not built from these sources)"

var (
	composeFile  string
	cliArtifacts []string
	skipBuild    bool
	force        bool
)

// NewCmdInit describes the CLI command to generate a skaffold configuration.
func NewCmdInit(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Automatically generate skaffold configuration for deploying an application",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doInit(out)
		},
	}
	cmd.Flags().StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip generating build artifacts in skaffold config")
	cmd.Flags().BoolVar(&force, "force", false, "Force the generation of the skaffold config")
	cmd.Flags().StringVar(&composeFile, "compose-file", "", "Initialize from a docker-compose file")
	cmd.Flags().StringArrayVarP(&cliArtifacts, "artifact", "a", nil, "'='-delimited dockerfile/image pair to generate build artifact\n(example: --artifact=/web/Dockerfile.web=gcr.io/web-project/image)")
	return cmd
}

func doInit(out io.Writer) error {
	rootDir := "."

	if composeFile != "" {
		// run kompose first to generate k8s manifests, then run skaffold init
		logrus.Infof("running 'kompose convert' for file %s", composeFile)
		komposeCmd := exec.Command("kompose", "convert", "-f", composeFile)
		if err := util.RunCmd(komposeCmd); err != nil {
			return errors.Wrap(err, "running kompose")
		}
	}

	var potentialConfigs, k8sConfigs, dockerfiles, images []string
	err := filepath.Walk(rootDir, func(path string, f os.FileInfo, e error) error {
		if f.IsDir() {
			return nil
		}
		if strings.HasPrefix(path, ".") {
			return nil
		}
		if util.IsSupportedKubernetesFormat(path) {
			potentialConfigs = append(potentialConfigs, path)
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

	for _, file := range potentialConfigs {
		if !force {
			config, err := schema.ParseConfig(file, false)
			if err == nil && config != nil {
				return fmt.Errorf("pre-existing %s found", file)
			}
		}

		logrus.Debugf("%s is not a valid skaffold configuration: continuing", file)
		imgs, err := parseKubernetesYaml(file)
		if err == nil {
			logrus.Infof("found valid k8s yaml: %s", file)
			k8sConfigs = append(k8sConfigs, file)
			images = append(images, imgs...)
		} else {
			logrus.Infof("invalid k8s yaml %s: %s", file, err.Error())
		}
	}

	var pairs []dockerfilePair
	// conditionally generate build artifacts
	if !skipBuild {
		if len(dockerfiles) == 0 {
			return errors.New("one or more valid Dockerfiles must be present to run skaffold; please provide at least one Dockerfile and try again")
		}

		if len(k8sConfigs) == 0 {
			return errors.New("one or more valid kubernetes manifests is required to run skaffold")
		}

		if cliArtifacts != nil {
			pairs, err = processCliArtifacts(cliArtifacts)
			if err != nil {
				return errors.Wrap(err, "processing cli artifacts")
			}
		} else {
			pairs = resolveDockerfileImages(dockerfiles, images)
		}
	}

	pipeline, err := generateSkaffoldPipeline(k8sConfigs, pairs)
	if err != nil {
		return err
	}

	if opts.ConfigurationFile == "-" {
		out.Write(pipeline)
		return nil
	}

	if !force {
		fmt.Fprintln(out, string(pipeline))

		reader := bufio.NewReader(os.Stdin)
	confirmLoop:
		for {
			fmt.Fprintf(out, "Do you want to write this configuration to %s? [y/n]: ", opts.ConfigurationFile)

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

	if err := ioutil.WriteFile(opts.ConfigurationFile, pipeline, 0644); err != nil {
		return errors.Wrap(err, "writing config to file")
	}

	fmt.Fprintf(out, "Configuration %s was written\n", opts.ConfigurationFile)
	tips.PrintForInit(out, opts)

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

func generateSkaffoldPipeline(k8sConfigs []string, dockerfilePairs []dockerfilePair) ([]byte, error) {
	// if we're here, the user has no skaffold yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	logrus.Info("generating skaffold config")

	pipeline := &latest.SkaffoldPipeline{
		APIVersion: latest.Version,
		Kind:       "Config",
	}
	if err := pipeline.SetDefaultValues(); err != nil {
		return nil, errors.Wrap(err, "generating default pipeline")
	}

	pipeline.Build = processBuildArtifacts(dockerfilePairs)
	pipeline.Deploy = latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: k8sConfigs,
			},
		},
	}

	pipelineStr, err := yaml.Marshal(pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling generated pipeline")
	}

	return pipelineStr, nil
}

// parseKubernetesYaml attempts to parse k8s objects from a yaml file
// if successful, it will return the images referenced in the k8s config
// so they can be built by the generated skaffold yaml
func parseKubernetesYaml(filepath string) ([]string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "opening config file")
	}
	r := k8syaml.NewYAMLReader(bufio.NewReader(f))

	objects := []runtime.Object{}
	images := []string{}

	for {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "reading config file")
		}
		d := scheme.Codecs.UniversalDeserializer()
		obj, _, err := d.Decode(doc, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "decoding kubernetes yaml")
		}

		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(doc, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		images = append(images, parseImagesFromYaml(m)...)
		objects = append(objects, obj)
	}
	if len(objects) == 0 {
		return nil, errors.New("no valid kubernetes objects decoded")
	}
	return images, nil
}

// adapted from pkg/skaffold/deploy/kubectl/recursiveReplaceImage()
func parseImagesFromYaml(doc interface{}) []string {
	images := []string{}
	switch t := doc.(type) {
	case []interface{}:
		for _, v := range t {
			images = append(images, parseImagesFromYaml(v)...)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			if k.(string) != "image" {
				images = append(images, parseImagesFromYaml(v)...)
				continue
			}

			images = append(images, v.(string))
		}
	}
	return images
}

type dockerfilePair struct {
	Dockerfile string
	ImageName  string
}
