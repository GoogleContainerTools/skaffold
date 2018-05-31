/*
Copyright 2018 The Skaffold Authors

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

package util

import (
	"testing"
	"text/template"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEnvTemplate_ExecuteEnvTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		customMap map[string]string
		env       []string
		want      string
		shouldErr bool
	}{
		{
			name:     "custom only",
			template: "{{.FOO}}:{{.BAR}}",
			customMap: map[string]string{
				"FOO": "foo",
				"BAR": "bar",
			},
			want: "foo:bar",
		},
		{
			name:     "env only",
			template: "{{.FOO}}-{{.BAZ}}:latest",
			env:      []string{"FOO=BAR", "BAZ=BAT"},
			want:     "BAR-BAT:latest",
		},
		{
			name:     "both and custom precedence",
			template: "{{.MY_NAME}}-{{.FROM_ENV}}:latest",
			env:      []string{"FROM_ENV=FOO", "MY_NAME=BAR"},
			customMap: map[string]string{
				"FOO":     "foo",
				"MY_NAME": "from_custom",
			},
			want: "from_custom-FOO:latest",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testTemplate := template.Must(template.New("").Parse(test.template))
			OSEnviron = func() []string {
				return test.env
			}

			got, err := ExecuteEnvTemplate(testTemplate, test.customMap)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.want, got)
		})
	}
}
