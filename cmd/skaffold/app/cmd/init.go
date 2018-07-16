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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	yaml "gopkg.in/yaml.v2"

	cmdutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	dockerParse "github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"k8s.io/apimachinery/pkg/runtime"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func NewCmdInit(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Automatically generate skaffold configuration for deploying an application",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doInit(out)
		},
	}
	AddRunDevFlags(cmd)
	return cmd
}

func doInit(out io.Writer) error {
	rootDir := "."
	yamlFiles := []string{}
	k8sConfigs := []string{}
	dockerfiles := []string{}
	images := []string{}
	err := filepath.Walk(rootDir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
			yamlFiles = append(yamlFiles, path)
		}
		// try and parse dockerfile
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "reading potential dockerfile")
		}
		instructions, err := dockerParse.Parse(b)
		if err == nil && len(instructions) > 0 {
			logrus.Infof("existing dockerfile found: %s", path)
			dockerfiles = append(dockerfiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, file := range yamlFiles {
		config, err := cmdutil.ParseConfig(file)
		if err == nil && config != nil {
			out.Write([]byte(fmt.Sprintf("pre-existing skaffold yaml %s found: exiting\n", file)))
			return nil
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
	if len(dockerfiles) == 0 {
		return errors.New("one or more valid Dockerfiles must be present to run skaffold; please provide at least one Dockerfile and try again")
	}

	if len(k8sConfigs) == 0 {
		return errors.New("one or more valid kubernetes configs is required to run skaffold")
	}

	pairs, err := resolveDockerfileImages(dockerfiles, images)
	if err != nil {
		return errors.Wrap(err, "resolving dockerfile/image pairs")
	}

	cfg, err := generateSkaffoldConfig(k8sConfigs, pairs)
	if err != nil {
		return err
	}
	out.Write(cfg)

	return nil
}

// For each image parsed from all k8s yamls, prompt the user for the
// Dockerfile that builds the referenced image
func resolveDockerfileImages(dockerfiles []string, images []string) ([]dockerfilePair, error) {
	// if we only have 1 image and 1 dockerfile, don't bother prompting
	if len(images) == 1 && len(dockerfiles) == 1 {
		return []dockerfilePair{{
			Dockerfile: dockerfiles[0],
			ImageName:  images[0],
		}}, nil
	}
	pairs := []dockerfilePair{}
	for _, image := range images {
		var selectedDockerfile string
		prompt := &survey.Select{
			Message: fmt.Sprintf("Choose the dockerfile to build image %s", image),
			Options: dockerfiles,
		}
		survey.AskOne(prompt, &selectedDockerfile, nil)
		pairs = append(pairs, dockerfilePair{
			Dockerfile: selectedDockerfile,
			ImageName:  image,
		})
		dockerfiles = util.RemoveFromSlice(dockerfiles, selectedDockerfile)
	}
	return pairs, nil
}

func generateSkaffoldConfig(k8sConfigs []string, dockerfilePairs []dockerfilePair) ([]byte, error) {
	// if we're here, the user has no skaffold yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	logrus.Info("generating skaffold config")

	var err error

	config, err := config.NewConfig()
	if err != nil {
		return nil, errors.Wrap(err, "generating default config")
	}

	var artifacts []*v1alpha2.Artifact
	for _, pair := range dockerfilePairs {
		artifacts = append(artifacts, &v1alpha2.Artifact{
			ImageName: pair.ImageName,
			Workspace: pair.Dockerfile,
		})
	}
	config.Build.Artifacts = artifacts

	config.Deploy = v1alpha2.DeployConfig{
		DeployType: v1alpha2.DeployType{
			KubectlDeploy: &v1alpha2.KubectlDeploy{
				Manifests: k8sConfigs,
			},
		},
	}

	cfgStr, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling generated config")
	}

	return cfgStr, nil
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
