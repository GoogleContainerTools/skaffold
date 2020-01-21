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
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var requiredFields = []string{"apiVersion", "kind", "metadata"}

// kubectl implements deploymentInitializer for the kubectl deployer.
type kubectl struct {
	configs []string
	images  []string
}

// kubectlAnalyzer is a Visitor during the directory analysis that collects kubernetes manifests
type kubectlAnalyzer struct {
	directoryAnalyzer
	kubernetesManifests []string
}

func (a *kubectlAnalyzer) analyzeFile(filePath string) error {
	if IsKubernetesManifest(filePath) && !isSkaffoldConfig(filePath) {
		a.kubernetesManifests = append(a.kubernetesManifests, filePath)
	}
	return nil
}

// newKubectlInitializer returns a kubectl skaffold generator.
func newKubectlInitializer(potentialConfigs []string) (*kubectl, error) {
	var k8sConfigs, images []string
	for _, file := range potentialConfigs {
		imgs, err := parseImagesFromKubernetesYaml(file)
		if err == nil {
			k8sConfigs = append(k8sConfigs, file)
			images = append(images, imgs...)
		}
	}
	if len(k8sConfigs) == 0 {
		return nil, errors.New("one or more valid Kubernetes manifests is required to run skaffold")
	}
	return &kubectl{
		configs: k8sConfigs,
		images:  images,
	}, nil
}

// IsKubernetesManifest is for determining if a file is a valid Kubernetes manifest
func IsKubernetesManifest(file string) bool {
	if !util.HasKubernetesFileExtension(file) {
		return false
	}

	_, err := parseKubernetesObjects(file)
	return err == nil
}

// deployConfig implements the Initializer interface and generates
// skaffold kubectl deployment config.
func (k *kubectl) deployConfig() latest.DeployConfig {
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
func (k *kubectl) GetImages() []string {
	return k.images
}

type yamlObject map[interface{}]interface{}

// parseImagesFromKubernetesYaml parses the kubernetes yamls, and if it finds at least one
// valid Kubernetes object, it will return the images referenced in them.
func parseImagesFromKubernetesYaml(filepath string) ([]string, error) {
	k8sObjects, err := parseKubernetesObjects(filepath)
	if err != nil {
		return nil, err
	}

	images := []string{}
	for _, k8sObject := range k8sObjects {
		images = append(images, parseImagesFromYaml(k8sObject)...)
	}
	return images, nil
}

// parseKubernetesObjects uses required fields from the k8s spec
// to determine if a provided yaml file is a valid k8s manifest, as detailed in
// https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields.
// If so, it will return the parsed objects.
func parseKubernetesObjects(filepath string) ([]yamlObject, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "opening config file")
	}
	r := k8syaml.NewYAMLReader(bufio.NewReader(f))

	var k8sObjects []yamlObject

	for {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "reading config file")
		}

		obj := make(yamlObject)
		if err := yaml.Unmarshal(doc, &obj); err != nil {
			return nil, errors.Wrap(err, "reading Kubernetes YAML")
		}

		if !hasRequiredK8sManifestFields(obj) {
			continue
		}

		k8sObjects = append(k8sObjects, obj)
	}
	if len(k8sObjects) == 0 {
		return nil, errors.New("no valid Kubernetes objects decoded")
	}
	return k8sObjects, nil
}

func hasRequiredK8sManifestFields(doc map[interface{}]interface{}) bool {
	for _, field := range requiredFields {
		if _, ok := doc[field]; !ok {
			logrus.Debugf("%s not present in yaml, continuing", field)
			return false
		}
	}
	return true
}

// adapted from pkg/skaffold/deploy/kubectl/recursiveReplaceImage()
func parseImagesFromYaml(obj interface{}) []string {
	images := []string{}
	switch t := obj.(type) {
	case []interface{}:
		for _, v := range t {
			images = append(images, parseImagesFromYaml(v)...)
		}
	case yamlObject:
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
