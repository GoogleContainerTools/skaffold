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

package runner

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDefaultLabeller(t *testing.T) {
	var tests = []struct {
		description string
		version     string
		expected    string
	}{
		{
			description: "version mentioned",
			version:     "1.0",
			expected:    "skaffold-1.0",
		},
		{
			description: "empty version should add postfix unknown",
			expected:    fmt.Sprintf("skaffold-unknown"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			l := &DefaultLabeller{
				version: test.version,
			}
			labels := l.Labels()

			expected := map[string]string{"app.kubernetes.io/managed-by": test.expected}
			t.CheckDeepEqual(expected, labels)
		})
	}
}

func TestK8sMangedByLabel(t *testing.T) {
	defaultLabeller := &DefaultLabeller{
		version: "version",
	}
	expected := "app.kubernetes.io/managed-by=skaffold-version"
	actual := defaultLabeller.K8sMangedByLabel()
	if actual != expected {
		t.Fatalf("actual label not equal to expected label. Actual: \n %s \n Expected: \n %s", actual, expected)
	}
}
