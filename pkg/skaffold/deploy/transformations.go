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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
)

type Registries struct {
	InsecureRegistries   map[string]bool
	DebugHelpersRegistry string
}

type ManifestTransform func(l kubectl.ManifestList, builds []build.Artifact, registries Registries) (kubectl.ManifestList, error)

// Transforms are applied to manifests
var manifestTransforms []ManifestTransform

// AddManifestTransform adds a transform to be applied when deploying.
func AddManifestTransform(newTransform ManifestTransform) {
	manifestTransforms = append(manifestTransforms, newTransform)
}
