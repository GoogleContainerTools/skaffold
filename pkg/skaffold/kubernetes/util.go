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

package kubernetes

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

type yamlObject map[string]interface{}

// These are the required fields for a yaml document to be a valid Kubernetes yaml
var requiredFields = []string{"apiVersion", "kind", "metadata"}

// These are the supported file formats for Kubernetes manifests
var validSuffixes = []string{".yml", ".yaml", ".json"}

// HasKubernetesFileExtension is for determining if a file under a glob pattern
// is deployable file format. It makes no attempt to check whether or not the file
// is actually deployable or has the correct contents.
func HasKubernetesFileExtension(n string) bool {
	for _, s := range validSuffixes {
		if strings.HasSuffix(n, s) {
			return true
		}
	}
	return false
}

// IsKubernetesManifest is for determining if a file is a valid Kubernetes manifest
func IsKubernetesManifest(file string) bool {
	if !HasKubernetesFileExtension(file) {
		return false
	}

	_, err := parseKubernetesObjects(file)
	return err == nil
}

// ParseImagesFromKubernetesYaml parses the kubernetes yamls, and if it finds at least one
// valid Kubernetes object, it will return the images referenced in them.
func ParseImagesFromKubernetesYaml(filepath string) ([]string, error) {
	k8sObjects, err := parseKubernetesObjects(filepath)
	if err != nil {
		return nil, err
	}

	var images []string
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
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	r := k8syaml.NewYAMLReader(bufio.NewReader(f))

	var k8sObjects []yamlObject

	for {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}

		obj := make(yamlObject)
		if err := yaml.Unmarshal(doc, &obj); err != nil {
			return nil, fmt.Errorf("reading Kubernetes YAML: %w", err)
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

func hasRequiredK8sManifestFields(doc map[string]interface{}) bool {
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
	var images []string

	switch t := obj.(type) {
	case []interface{}:
		for _, v := range t {
			images = append(images, parseImagesFromYaml(v)...)
		}
	case yamlObject:
		for k, v := range t {
			if k != "image" {
				images = append(images, parseImagesFromYaml(v)...)
				continue
			}

			if value, ok := v.(string); ok {
				images = append(images, value)
			}
		}
	}

	return images
}
