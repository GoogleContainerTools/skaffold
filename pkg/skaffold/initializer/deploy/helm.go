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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// helm implements deploymentInitializer for the kustomize deployer.
type helm struct {
	chartPaths []string
	images     []string
}

// newHelmInitializer returns a helm config generator.
func newHelmInitializer(charts []string) *helm {
	var images []string
	for _, file := range charts {
		imgs, err := kubernetes.ParseImagesFromKubernetesYaml(file)
		if err == nil {
			images = append(images, imgs...)
		}
	}
	return &helm{
		chartPaths: charts,
		images:     images,
	}
}

// DeployConfig implements the Initializer interface and generates
// a helm configuration
func (h *helm) DeployConfig() (latestV1.DeployConfig, []latestV1.Profile) {

	return latestV1.DeployConfig{
		DeployType: latestV1.DeployType{},
	}, nil
}

// GetImages implements the Initializer interface and lists all the
// images present in the k8s manifest files.
func (h *helm) GetImages() []string {
	return []string{}
}

// Validate implements the Initializer interface and ensures
// we have at least one manifest before generating a config
func (h *helm) Validate() error {
	if len(h.chartPaths) == 0 {
		return errors.NoManifestErr{}
	}
	return nil
}

// we don't generate k8s manifests for a kustomize deploy
func (h *helm) AddManifestForImage(string, string) {}
