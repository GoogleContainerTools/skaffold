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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// deploymentInitializer detects a deployment type and is able to extract image names from it
type DeploymentInitializer interface {
	// deployConfig generates Deploy Config for skaffold configuration.
	DeployConfig() latest.DeployConfig
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
}

type cliDeployInit struct {
	cliKubernetesManifests []string
}

func (c *cliDeployInit) DeployConfig() latest.DeployConfig {
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: c.cliKubernetesManifests,
			}},
	}
}

func (c *cliDeployInit) GetImages() []string {
	return nil
}

type emptyDeployInit struct {
}

func (c *emptyDeployInit) DeployConfig() latest.DeployConfig {
	return latest.DeployConfig{}
}

func (c *emptyDeployInit) GetImages() []string {
	return nil
}

func NewDeployInitializer(manifests []string, c config.Config) (DeploymentInitializer, error) {
	switch {
	case c.SkipDeploy:
		return &emptyDeployInit{}, nil
	case len(c.CliKubernetesManifests) > 0:
		return &cliDeployInit{c.CliKubernetesManifests}, nil
	default:
		return newKubectlInitializer(manifests)
	}
}
