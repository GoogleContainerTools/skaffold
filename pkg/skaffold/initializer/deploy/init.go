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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Initializer detects a deployment type and is able to extract image names from it
type Initializer interface {
	// deployConfig generates Deploy Config for skaffold configuration.
	DeployConfig() (latest.DeployConfig, []latest.Profile)
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
	// Validate ensures preconditions are met before generating a skaffold config
	Validate() error
	// AddManifestForImage adds a provided manifest for a given image to the initializer
	AddManifestForImage(string, string)
}

type cliDeployInit struct {
	cliKubernetesManifests []string
}

func (c *cliDeployInit) DeployConfig() (latest.DeployConfig, []latest.Profile) {
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			KubectlDeploy: &latest.KubectlDeploy{
				Manifests: c.cliKubernetesManifests,
			},
		},
	}, nil
}

func (c *cliDeployInit) GetImages() []string {
	return nil
}

func (c *cliDeployInit) Validate() error {
	if len(c.cliKubernetesManifests) == 0 {
		return errors.NoManifestErr{}
	}
	return nil
}

func (c *cliDeployInit) AddManifestForImage(string, string) {}

type emptyDeployInit struct {
}

func (e *emptyDeployInit) DeployConfig() (latest.DeployConfig, []latest.Profile) {
	return latest.DeployConfig{}, nil
}

func (e *emptyDeployInit) GetImages() []string {
	return nil
}

func (e *emptyDeployInit) Validate() error {
	return nil
}

func (e *emptyDeployInit) AddManifestForImage(string, string) {}

// if any CLI manifests are provided, we always use those as part of a kubectl deploy first
// if not, then if a kustomization yaml is found, we use that next
// otherwise, default to a kubectl deploy.
func NewInitializer(manifests, bases, kustomizations []string, c config.Config) Initializer {
	switch {
	case c.SkipDeploy:
		return &emptyDeployInit{}
	case len(c.CliKubernetesManifests) > 0:
		return &cliDeployInit{c.CliKubernetesManifests}
	case len(kustomizations) > 0:
		return newKustomizeInitializer(c.DefaultKustomization, bases, kustomizations, manifests)
	default:
		return newKubectlInitializer(manifests)
	}
}
