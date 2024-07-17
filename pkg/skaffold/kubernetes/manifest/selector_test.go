/*
Copyright 2023 The Skaffold Authors

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
	"encoding/json"
	"regexp"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestWildcardGroupKindUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		expected WildcardGroupKind
	}{
		{
			name:     "empty JSON",
			data:     "{}",
			expected: WildcardGroupKind{},
		},
		{
			name:     "JSON err",
			data:     "{",
			expected: WildcardGroupKind{},
			wantErr:  true,
		},
		{
			name: "JSON with group and kind",
			data: `{"group": "core", "kind": "ConfigMap"}`,
			expected: WildcardGroupKind{
				Group: regexp.MustCompile("core"),
				Kind:  regexp.MustCompile("ConfigMap"),
			},
		},
		{
			name: "JSON with group",
			data: `{"group": "core" }`,
			expected: WildcardGroupKind{
				Group: regexp.MustCompile("core"),
			},
		},
		{
			name: "JSON with kind",
			data: `{"kind": "ConfigMap"}`,
			expected: WildcardGroupKind{
				Kind: regexp.MustCompile("ConfigMap"),
			},
		},
	}
	for _, tt := range tests {
		testutil.Run(t, tt.name, func(t *testutil.T) {
			var w WildcardGroupKind
			err := json.Unmarshal([]byte(tt.data), &w)
			t.CheckError(tt.wantErr, err)
			if tt.expected.Group == nil {
				t.CheckNil(w.Group)
			} else {
				t.CheckDeepEqual(w.Group.String(), tt.expected.Group.String())
			}

			if tt.expected.Kind == nil {
				t.CheckNil(w.Kind)
			} else {
				t.CheckDeepEqual(w.Kind.String(), tt.expected.Kind.String())
			}
		})
	}
}
