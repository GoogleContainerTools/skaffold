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
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

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
	if kubernetes.IsKubernetesManifest(filePath) && !schema.IsSkaffoldConfig(filePath) {
		a.kubernetesManifests = append(a.kubernetesManifests, filePath)
	}
	return nil
}

// newKubectlInitializer returns a kubectl skaffold generator.
func newKubectlInitializer(potentialConfigs []string) (*kubectl, error) {
	var k8sConfigs, images []string
	for _, file := range potentialConfigs {
		imgs, err := kubernetes.ParseImagesFromKubernetesYaml(file)
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
