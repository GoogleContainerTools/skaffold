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

package labels

import (
	"testing"

	"github.com/google/go-github/github"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateLabelsFromPR(t *testing.T) {
	want := map[string]string{
		"docs-controller-deployment": "true",
		"deployment":                 "docs-controller-deployment-1",
	}
	got := GenerateLabelsFromPR(1)

	testutil.CheckDeepEqual(t, want, got)
}

func TestSelector(t *testing.T) {
	want := "deployment=docs-controller-deployment-1"
	got := Selector(1)

	testutil.CheckDeepEqual(t, want, got)
}

func TestRetrieveLabel(t *testing.T) {
	wantKey, wantValue := "deployment", "docs-controller-deployment-1"
	gotKey, gotValue := RetrieveLabel(1)

	testutil.CheckDeepEqual(t, wantKey, gotKey)
	testutil.CheckDeepEqual(t, wantValue, gotValue)
}

func TestDocsLabelExists(t *testing.T) {
	tests := []struct {
		description string
		labels      []*github.Label
		expected    bool
	}{
		{
			description: "labels are nil",
			labels:      nil,
			expected:    false,
		},
		{
			description: "labels are empty",
			labels:      []*github.Label{},
			expected:    false,
		},
		{
			description: "doesn't contain the right label",
			labels:      []*github.Label{nil, {Name: github.String("test")}},
			expected:    false,
		},
		{
			description: "contains the right label",
			labels:      []*github.Label{{Name: github.String("test")}, {Name: github.String(constants.DocsLabel)}},
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got := DocsLabelExists(test.labels)

			t.CheckDeepEqual(test.expected, got)
		})
	}
}
