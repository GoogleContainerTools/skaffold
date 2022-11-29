/*
Copyright 2022 The Skaffold Authors

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

package render

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// kubectl implements render Initializer for the kubectl renderer.
type kubectl struct {
	configs []string // the k8s manifest files present in the project
	images  []string // the images parsed from the k8s manifest files
}

// newKubectlInitializer returns a kubectl skaffold generator.
func newKubectlInitializer(potentialConfigs []string) *kubectl {
	var k8sConfigs, images []string
	for _, file := range potentialConfigs {
		imgs, err := kubernetes.ParseImagesFromKubernetesYaml(file)
		if err == nil {
			k8sConfigs = append(k8sConfigs, file)
			images = append(images, imgs...)
		}
	}
	return &kubectl{
		configs: k8sConfigs,
		images:  images,
	}
}

// RenderConfig implements the Initializer interface and generates
// skaffold kubectl render config.
func (k *kubectl) RenderConfig() (latest.RenderConfig, []latest.Profile) {
	return latest.RenderConfig{
		Generate: latest.Generate{
			RawK8s: k.configs,
		},
	}, nil
}

// GetImages implements the Initializer interface and lists all the
// images present in the k8s manifest files.
func (k *kubectl) GetImages() []string {
	return k.images
}

// Validate implements the Initializer interface and ensures
// we have at least one manifest before generating a config
func (k *kubectl) Validate() error {
	if len(k.images) == 0 {
		return errors.NoManifestErr{}
	}
	return nil
}

func (k *kubectl) AddManifestForImage(path, image string) {
	k.configs = append(k.configs, path)
	k.images = append(k.images, image)
}
