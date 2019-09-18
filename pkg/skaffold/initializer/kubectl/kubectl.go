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

package kubectl

import (
	"bufio"
	"io"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// ValidSuffixes are the supported file formats for kubernetes manifests
var ValidSuffixes = []string{".yml", ".yaml", ".json"}

var requiredFields = []string{"apiVersion", "kind", "metadata"}

// Kubectl holds parameters to run kubectl.
type Kubectl struct {
	configs []string
	images  []string
}

// New returns a Kubectl skaffold generator.
func New(potentialConfigs []string) (*Kubectl, error) {
	var k8sConfigs, images []string
	for _, file := range potentialConfigs {
		imgs, err := parseImagesFromKubernetesYaml(file)
		if err == nil {
			logrus.Infof("found valid k8s yaml: %s", file)
			k8sConfigs = append(k8sConfigs, file)
			images = append(images, imgs...)
		} else {
			logrus.Infof("invalid k8s yaml %s: %s", file, err.Error())
		}
	}
	if len(k8sConfigs) == 0 {
		return nil, errors.New("one or more valid kubernetes manifests is required to run skaffold")
	}
	return &Kubectl{
		configs: k8sConfigs,
		images:  images,
	}, nil
}

// GenerateDeployConfig implements the Initializer interface and generates
// skaffold kubectl deployment config.
func (k *Kubectl) GenerateDeployConfig() latest.DeployConfig {
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: k.configs,
			},
		},
	}
}

// GetImages implements the Initializer interface and lists all the
// images present in the k8 manifest files.
func (k *Kubectl) GetImages() []string {
	return k.images
}

// parseImagesFromKubernetesYaml uses required fields from the k8s spec
// to determine if a provided yaml file is a valid k8s manifest, as detailed in
// https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields.
// if so, it will return the images referenced in the k8s config
// so they can be built by the generated skaffold yaml
func parseImagesFromKubernetesYaml(filepath string) ([]string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "opening config file")
	}
	r := k8syaml.NewYAMLReader(bufio.NewReader(f))

	yamlsFound := 0
	images := []string{}

	for {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "reading config file")
		}

		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(doc, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if !isKubernetesYaml(m) {
			continue
		}

		yamlsFound++

		images = append(images, parseImagesFromYaml(m)...)
	}
	if yamlsFound == 0 {
		return nil, errors.New("no valid kubernetes objects decoded")
	}
	return images, nil
}

func isKubernetesYaml(doc map[interface{}]interface{}) bool {
	for _, field := range requiredFields {
		if _, ok := doc[field]; !ok {
			logrus.Debugf("%s not present in yaml, continuing", field)
			return false
		}
	}
	return true
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
