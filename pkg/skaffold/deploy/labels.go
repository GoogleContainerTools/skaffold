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
	"k8s.io/apimachinery/pkg/runtime"
)

// Artifact contains all information about a completed deployment
type Artifact struct {
	Obj       runtime.Object
	Namespace string
}

// Labeller can give key/value labels to set on deployed resources.
type Labeller interface {
	// Labels keys must be prefixed with "skaffold.dev/"
	Labels() map[string]string
}

// merge merges the labels from multiple sources.
func merge(sources ...Labeller) map[string]string {
	merged := make(map[string]string)

	for _, src := range sources {
		copyMap(merged, src.Labels())
	}

	return merged
}

func copyMap(dest, from map[string]string) {
	for k, v := range from {
		dest[k] = v
	}
}
