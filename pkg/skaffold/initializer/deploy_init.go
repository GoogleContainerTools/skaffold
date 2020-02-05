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

package initializer

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Error string

func (e Error) Error() string { return string(e) }

const NoManifest = Error("one or more Kubernetes manifests is required to run skaffold")

// deploymentInitializer detects a deployment type and is able to extract image names from it
type deploymentInitializer interface {
	// deployConfig generates Deploy Config for skaffold configuration.
	deployConfig() latest.DeployConfig
	// AddManifest adds a path to the list of manifest paths for a given image
	AddManifestForImage(string, string)
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
	// Validate ensures preconditions are met before generating a skaffold config
	Validate() error
}

type cliDeployInit struct {
	cliKubernetesManifests []string
}

func (c *cliDeployInit) deployConfig() latest.DeployConfig {
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: c.cliKubernetesManifests,
			}},
	}
}

func (c *cliDeployInit) AddManifestForImage(string, string) {}

func (c *cliDeployInit) GetImages() []string {
	return nil
}

func (c *cliDeployInit) Validate() error {
	if len(c.cliKubernetesManifests) == 0 {
		return NoManifest
	}
	return nil
}

type emptyDeployInit struct {
}

func (c *emptyDeployInit) deployConfig() latest.DeployConfig {
	return latest.DeployConfig{}
}

func (c *emptyDeployInit) AddManifestForImage(string, string) {}

func (c *emptyDeployInit) GetImages() []string {
	return nil
}

func (c *emptyDeployInit) Validate() error {
	return nil
}
