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

package kubectl

import (
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Replacer is used to replace portions of yaml manifests that match a given key.
type Replacer interface {
	Matches(key string) bool

	NewValue(old interface{}) (bool, interface{})
}

// Visit recursively visits a list of manifests and applies transformations of them.
func (l *ManifestList) Visit(replacer Replacer) (ManifestList, error) {
	var updated ManifestList

	for _, manifest := range *l {
		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(manifest, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if len(m) == 0 {
			continue
		}

		recursiveVisit(m, replacer)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling yaml")
		}

		updated = append(updated, updatedManifest)
	}

	return updated, nil
}

func recursiveVisit(i interface{}, replacer Replacer) {
	switch t := i.(type) {
	case []interface{}:
		for _, v := range t {
			recursiveVisit(v, replacer)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			key := k.(string)

			if !replacer.Matches(key) {
				recursiveVisit(v, replacer)
				continue
			}

			ok, newValue := replacer.NewValue(v)
			if ok {
				t[k] = newValue
			}
		}
	}
}
