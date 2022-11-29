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
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// Initializer detects a deployment type and is able to extract image names from it
type Initializer interface {
	// renderConfig generates Render Config for skaffold configuration.
	RenderConfig() (latest.RenderConfig, []latest.Profile)
	// GetImages fetches all the images defined in the manifest files.
	GetImages() []string
	// Validate ensures preconditions are met before generating a skaffold config
	Validate() error
	// AddManifestForImage adds a provided manifest for a given image to the initializer
	AddManifestForImage(string, string)
}

type cliManifestsInit struct {
	cliKubernetesManifests []string
}

func (c *cliManifestsInit) RenderConfig() (latest.RenderConfig, []latest.Profile) {
	return latest.RenderConfig{
		Generate: latest.Generate{RawK8s: c.cliKubernetesManifests},
	}, nil
}

func (c *cliManifestsInit) GetImages() []string {
	return nil
}

func (c *cliManifestsInit) Validate() error {
	if len(c.cliKubernetesManifests) == 0 {
		return errors.NoManifestErr{}
	}
	return nil
}

func (c *cliManifestsInit) AddManifestForImage(string, string) {}

type emptyRenderInit struct {
}

func (e *emptyRenderInit) RenderConfig() (latest.RenderConfig, []latest.Profile) {
	return latest.RenderConfig{}, nil
}

func (e *emptyRenderInit) GetImages() []string {
	return nil
}

func (e *emptyRenderInit) Validate() error {
	return nil
}

func (e *emptyRenderInit) AddManifestForImage(string, string) {}

// if any CLI manifests are provided, we always use those as part of a kubectl render first
// if not, then if a kustomization yaml is found, we use that next
// otherwise, default to a kubectl render.
func NewInitializer(manifests, bases, kustomizations []string, h analyze.HelmChartInfo, c config.Config) Initializer {
	switch {
	case c.SkipDeploy:
		return &emptyRenderInit{}
	case len(c.CliKubernetesManifests) > 0:
		return &cliManifestsInit{c.CliKubernetesManifests}
	case len(kustomizations) > 0:
		return newKustomizeInitializer(c.DefaultKustomization, bases, kustomizations, manifests)
	case len(h.Charts()) > 0:
		return newHelmInitializer(h.Charts())
	default:
		return newKubectlInitializer(manifests)
	}
}
