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
	"text/template"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestTemplateUtils_GetTemplateVariables(t *testing.T) {
	tests := []struct {
		description string
		template    string
		expected    []string
	}{
		{
			description: "variables",
			template:    "{{.V1}}  {{.V1}}    {{.V2}}",
			expected:    []string{"V1", "V2"},
		},
		{
			description: "condition",
			template:    "{{.V1}}{{if ne .V1 .V2}}{{.V3}}{{else if .V4}}{{else}}{{.V5}}{{end}}",
			expected:    []string{"V1", "V2", "V3", "V4", "V5"},
		},
		{
			description: "range",
			template:    "{{range .V1}}{{.V2}}{{end}}",
			expected:    []string{"V1", "V2"},
		},
		{
			description: "block-template",
			template:    `{{block "b1" .V1}}block content{{end}}{{template "t1" .V2}}`,
			expected:    []string{"V1", "V2"},
		},
		{
			description: "with",
			template:    "{{with .V1}} {{.V2}} {{else}} {{.V3}} {{end}}",
			expected:    []string{"V1", "V2", "V3"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			parsedTemplate, _ := template.New(test.description).Parse(test.template)
			variables := GetTemplateFields(parsedTemplate)

			t.CheckDeepEqual(test.expected, variables)
		})
	}
}
