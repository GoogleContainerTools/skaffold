/*
Copyright 2020 The Skaffold Authors

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// kustomize implements deploymentInitializer for the kustomize deployer.
type kustomize struct {
	kustomizations []string
	images         []string
}

// newKustomizeInitializer returns a kustomize config generator.
func newKustomizeInitializer(kustomizations []string, potentialConfigs []string) *kustomize {
	var images []string
	for _, file := range potentialConfigs {
		imgs, err := kubernetes.ParseImagesFromKubernetesYaml(file)
		if err == nil {
			images = append(images, imgs...)
		}
	}
	return &kustomize{
		images:         images,
		kustomizations: kustomizations,
	}
}

// deployConfig implements the Initializer interface and generates
// a kustomize deployment config.
func (k *kustomize) DeployConfig() latest.DeployConfig {
	var kustomizeConfig *latest.KustomizeDeploy
	// if we only have the default path, leave the config empty - it's cleaner
	if len(k.kustomizations) == 1 && k.kustomizations[0] == deploy.DefaultKustomizePath {
		kustomizeConfig = &latest.KustomizeDeploy{}
	} else {
		kustomizeConfig = &latest.KustomizeDeploy{
			KustomizePaths: k.kustomizations,
		}
	}
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			KustomizeDeploy: kustomizeConfig,
		},
	}
}

// GetImages implements the Initializer interface and lists all the
// images present in the k8s manifest files.
func (k *kustomize) GetImages() []string {
	return k.images
}

// Validate implements the Initializer interface and ensures
// we have at least one manifest before generating a config
func (k *kustomize) Validate() error {
	if len(k.images) == 0 {
		return NoManifest
	}
	return nil
}

// we don't generate k8s manifests for a kustomize deploy
func (k *kustomize) AddManifestForImage(string, string) {}
