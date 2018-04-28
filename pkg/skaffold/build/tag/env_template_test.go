/*
Copyright 2018 Google LLC

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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEnvTemplateTagger_GenerateFullyQualifiedImageName(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		opts      *TagOptions
		env       []string
		want      string
		shouldErr bool
	}{
		{
			name:     "empty env",
			template: "{{.IMAGE_NAME}}:{{.DIGEST}}",
			opts: &TagOptions{
				ImageName: "foo",
				Digest:    "bar",
			},
			want: "foo:bar",
		},
		{
			name:     "env",
			template: "{{.FOO}}-{{.BAZ}}:latest",
			env:      []string{"FOO=BAR", "BAZ=BAT"},
			opts: &TagOptions{
				ImageName: "foo",
				Digest:    "bar",
			},
			want: "BAR-BAT:latest",
		},
		{
			name:     "opts precedence",
			template: "{{.IMAGE_NAME}}-{{.FROM_ENV}}:latest",
			env:      []string{"FROM_ENV=FOO", "IMAGE_NAME=BAT"},
			opts: &TagOptions{
				ImageName: "image_name",
				Digest:    "bar",
			},
			want: "image_name-FOO:latest",
		},
		{
			name:     "digest algo hex",
			template: "{{.IMAGE_NAME}}:{{.DIGEST_ALGO}}-{{.DIGEST_HEX}}",
			opts: &TagOptions{
				ImageName: "foo",
				Digest:    "sha256:abcd",
			},
			want: "foo:sha256-abcd",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &EnvTemplateTagger{
				Template: template.Must(template.New("").Parse(test.template)),
			}
			environ = func() []string {
				return test.env
			}

			got, err := c.GenerateFullyQualifiedImageName("", test.opts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.want, got)
		})
	}
}

func TestNewEnvTemplateTagger(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		shouldErr bool
	}{
		{
			name:     "valid template",
			template: "{{.FOO}}",
		},
		{
			name:      "invalid template",
			template:  "{{.FOO",
			shouldErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnvTemplateTagger(tt.template)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}
