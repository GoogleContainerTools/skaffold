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

package manifest

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
)

type Registries struct {
	InsecureRegistries   map[string]bool
	DebugHelpersRegistry string
}

type Transform func(l ManifestList, builds []build.Artifact, registries Registries) (ManifestList, error)

// Transforms are applied to manifests
var transforms []Transform

// AddTransform adds a transform to be applied when deploying.
func AddTransform(newTransform Transform) {
	transforms = append(transforms, newTransform)
}

// GetTransforms returns all manifest transforms.
func GetTransforms() []Transform {
	return transforms
}

// ApplyTransforms applies all manifests transforms to the provided manifests.
func ApplyTransforms(manifests ManifestList, builds []build.Artifact, insecureRegistries map[string]bool, debugHelpersRegistry string) (ManifestList, error) {
	var err error
	for _, transform := range transforms {
		manifests, err = transform(manifests, builds, Registries{insecureRegistries, debugHelpersRegistry})
		if err != nil {
			return nil, transformManifestErr(err)
		}
	}
	return manifests, nil
}
