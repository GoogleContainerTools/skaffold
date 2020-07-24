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

package tag

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTagTemplate_ExecuteTagTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		customMap   map[string]string
		expected    string
		shouldErr   bool
	}{
		{
			description: "empty template",
		},
		{
			description: "only text",
			template:    "foo-bar",
			expected:    "foo-bar",
		},
		{
			description: "only component",
			template:    "{{.FOO}}",
			expected:    "2016-02-05",
			customMap:   map[string]string{"FOO": "2016-02-05"},
		},
		{
			description: "both text and components",
			template:    "foo-{{.BAR}}",
			expected:    "foo-2016-02-05",
			customMap:   map[string]string{"BAR": "2016-02-05"},
		},
		{
			description: "component has value with len 0",
			template:    "foo-{{.BAR}}",
			expected:    "foo-",
			customMap:   map[string]string{"BAR": ""},
		},
		{
			description: "undefined component",
			template:    "foo-{{.BAR}}",
			customMap:   map[string]string{"FOO": "2016-02-05"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testTemplate, err := ParseTagTemplate(test.template)
			t.CheckNoError(err)

			got, err := ExecuteTagTemplate(testTemplate.Option("missingkey=error"), test.customMap)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
		})
	}
}

func TestTagTemplate_ParseTagTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		shouldErr   bool
	}{
		{
			description: "valid template",
			template:    "{{.FOO}}",
		},
		{
			description: "invalid template",
			template:    "{{.FOO",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := ParseTagTemplate(test.template)
			t.CheckError(test.shouldErr, err)
		})
	}
}
