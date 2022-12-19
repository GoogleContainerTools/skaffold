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
	"context"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// kustomize implements deploymentInitializer for the kustomize deployer.
type kustomize struct {
	defaultKustomization string
	kustomizations       []string
	bases                []string
	images               []string
}

// newKustomizeInitializer returns a kustomize config generator.
func newKustomizeInitializer(defaultKustomization string, bases, kustomizations, potentialConfigs []string) *kustomize {
	var images []string
	for _, file := range potentialConfigs {
		imgs, err := kubernetes.ParseImagesFromKubernetesYaml(file)
		if err == nil {
			images = append(images, imgs...)
		}
	}
	return &kustomize{
		defaultKustomization: defaultKustomization,
		images:               images,
		bases:                bases,
		kustomizations:       kustomizations,
	}
}

// RenderConfig implements the Initializer interface and generates
// a kustomize deployment config.
func (k *kustomize) RenderConfig() (latest.RenderConfig, []latest.Profile) {
	var kustomizeConfig *latest.Kustomize
	var profiles []latest.Profile

	// if there's only one kustomize path, generate a config with that single path and return it
	if len(k.kustomizations) == 1 {
		kustomizeConfig = &latest.Kustomize{
			Paths: k.kustomizations,
		}
		return latest.RenderConfig{
			Generate: latest.Generate{
				Kustomize: kustomizeConfig,
			},
		}, nil
	}

	// if there are multiple paths, generate a config that chooses a default
	// kustomization based on our heuristic, and creates separate profiles
	// for all other overlays in the project
	defaultKustomization := k.defaultKustomization
	if defaultKustomization == "" {
		// either choose one that's called "dev", or else the first one that isn't called "prod"
		dev, prod := -1, -1
		for i, kustomization := range k.kustomizations {
			switch filepath.Base(kustomization) {
			case "dev":
				dev = i
			case "prod":
				prod = i
			default:
			}
		}

		switch {
		case dev != -1:
			defaultKustomization = k.kustomizations[dev]
		case prod == 0:
			defaultKustomization = k.kustomizations[1]
		default:
			defaultKustomization = k.kustomizations[0]
		}
		log.Entry(context.TODO()).Warnf("multiple kustomizations found but no default provided - defaulting to %s", defaultKustomization)
	}

	for _, kustomization := range k.kustomizations {
		if kustomization == defaultKustomization {
			kustomizeConfig = &latest.Kustomize{
				Paths: []string{defaultKustomization},
			}
		} else {
			profiles = append(profiles, latest.Profile{
				Name: filepath.Base(kustomization),
				Pipeline: latest.Pipeline{
					Render: latest.RenderConfig{
						Generate: latest.Generate{
							Kustomize: &latest.Kustomize{
								Paths: []string{kustomization},
							},
						},
					},
				},
			})
		}
	}

	return latest.RenderConfig{
		Generate: latest.Generate{
			Kustomize: kustomizeConfig,
		},
	}, profiles
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
		return errors.NoManifestErr{}
	}
	return nil
}

func (k *kustomize) AddManifestForImage(string, string) {}
