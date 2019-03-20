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
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// ValidSuffixes are the supported file formats for kubernetes manifests
var ValidSuffixes = []string{".yml", ".yaml", ".json"}

// Kubectl holds parameters to run kubectl.
type Kubectl struct {
	configs []string
	images  []string
}

//New returns a Kubectl skaffold generator.
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

// parseImagesFromKubernetesYaml attempts to parse k8s objects from a yaml file
// if successful, it will return the images referenced in the k8s config
// so they can be built by the generated skaffold yaml
func parseImagesFromKubernetesYaml(filepath string) ([]string, error) {
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
