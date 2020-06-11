/*
Copyright 2020 The Skaffold Authors

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

package diag

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/diag/validator"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockValidator struct {
	ns          []string
	listOptions metav1.ListOptions
}

func (m *mockValidator) Validate(_ context.Context, ns string, opts metav1.ListOptions) ([]validator.Resource, error) {
	m.ns = append(m.ns, ns)
	m.listOptions = opts
	return nil, nil
}

func TestRun(t *testing.T) {
	tests := []struct {
		description string
		labels      map[string]string
		ns          []string
		expected    *mockValidator
	}{
		{
			description: "multiple namespaces with an empty namespace and no labels",
			ns:          []string{"foo", "bar", ""},
			expected: &mockValidator{
				ns:          []string{"foo", "bar"},
				listOptions: metav1.ListOptions{},
			},
		},
		{
			description: "empty namespaces no labels",
			ns:          []string{""},
			expected:    &mockValidator{ns: nil},
		},
		{
			description: "multiple namespaces and multiple labels",
			ns:          []string{"foo", "goo"},
			labels: map[string]string{
				"skaffold":       "session",
				"deployment-app": "app",
			},
			expected: &mockValidator{
				ns: []string{"foo", "goo"},
				listOptions: metav1.ListOptions{
					LabelSelector: "deployment-app=app,skaffold=session",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			d := New(test.ns)
			for k, v := range test.labels {
				d = d.WithLabel(k, v)
			}
			m := &mockValidator{}
			d = d.WithValidators([]validator.Validator{m})
			d.Run(context.Background())
			t.CheckDeepEqual(test.expected, m, cmp.AllowUnexported(mockValidator{}))
		})
	}
}
