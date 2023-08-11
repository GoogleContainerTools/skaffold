package manifest

import (
	"encoding/json"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	"regexp"
	"testing"
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
