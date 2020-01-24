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

import "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"

// deploymentInitializer detects a deployment type and is able to extract image names from it
type deploymentInitializer interface {
	// deployConfig generates Deploy Config for skaffold configuration.
	deployConfig() latest.DeployConfig
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
}

type cliDeployInit struct {
	cliKubectlManifests []string
}

func (c *cliDeployInit) deployConfig() latest.DeployConfig {
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: c.cliKubectlManifests,
			}},
	}
}

func (c *cliDeployInit) GetImages() []string {
	return nil
}

type emptyDeployInit struct {
}

func (c *emptyDeployInit) deployConfig() latest.DeployConfig {
	return latest.DeployConfig{}
}

func (c *emptyDeployInit) GetImages() []string {
	return nil
}
