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

package deploy

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/generator"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// kubectl implements deploymentInitializer for the kubectl deployer.
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

// deployConfig implements the Initializer interface and generates
// skaffold kubectl deployment config.
func (k *kubectl) DeployConfig() latest.DeployConfig {
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

// Validate implements the Initializer interface and ensures
// we have at least one manifest before generating a config
func (k *kubectl) Validate() error {
	if len(k.images) == 0 {
		return NoManifest
	}
	return nil
}

// GenerateManifests implements the Initializer interface and
// generates manifests for each unresolved image
func (k *kubectl) GenerateManifests(unresolved []build.GeneratedBuilderImagePair) (map[string][]byte, error) {
	generatedManifests := map[string][]byte{}
	for _, pair := range unresolved {
		manifest, err := generator.Generate(pair.ImageName)
		if err != nil {
			return nil, errors.Wrap(err, "generating kubernetes manifest")
		}
		path := filepath.Join(pair.ManifestPath, "deployment.yaml")
		generatedManifests[path] = manifest
		k.addManifestForImage(path, pair.ImageName)
	}
	return generatedManifests, nil
}

func (k *kubectl) addManifestForImage(path, image string) {
	k.configs = append(k.configs, path)
	k.images = append(k.images, image)
}
