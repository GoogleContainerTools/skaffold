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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsInsecure(t *testing.T) {
	tests := []struct {
		description        string
		registry           string
		insecureRegistries map[string]bool
		result             bool
	}{
		{"nil registries", "localhost:5000", nil, false},
		{"unlisted registry", "other.tld", map[string]bool{"registry.tld": true}, false},
		{"listed insecure", "registry.tld", map[string]bool{"registry.tld": true}, true},
		{"listed secure", "registry.tld", map[string]bool{"registry.tld": false}, false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := IsInsecure(test.registry, test.insecureRegistries)

			t.CheckDeepEqual(test.result, result)
		})
	}
}
