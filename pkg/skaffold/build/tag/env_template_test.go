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
	type fields struct {
		Template string
	}
	type args struct {
		opts *TagOptions
		env  []string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      string
		shouldErr bool
	}{
		{
			name: "empty env",
			fields: fields{
				Template: "{{.IMAGE_NAME}}:{{.DIGEST}}",
			},
			args: args{
				opts: &TagOptions{
					ImageName: "foo",
					Digest:    "bar",
				},
			},
			want: "foo:bar",
		},
		{
			name: "env",
			fields: fields{
				Template: "{{.FOO}}-{{.BAZ}}:latest",
			},
			args: args{
				env: []string{"FOO=BAR", "BAZ=BAT"},
				opts: &TagOptions{
					ImageName: "foo",
					Digest:    "bar",
				},
			},
			want: "BAR-BAT:latest",
		},
		{
			name: "opts precedence",
			fields: fields{
				Template: "{{.IMAGE_NAME}}-{{.FROM_ENV}}:latest",
			},
			args: args{
				env: []string{"FROM_ENV=FOO", "IMAGE_NAME=BAT"},
				opts: &TagOptions{
					ImageName: "image_name",
					Digest:    "bar",
				},
			},
			want: "image_name-FOO:latest",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &EnvTemplateTagger{
				Template: template.Must(template.New("").Parse(test.fields.Template)),
			}
			environ = func() []string {
				return test.args.env
			}

			got, err := c.GenerateFullyQualifiedImageName("", test.args.opts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.want, got)
		})
	}
}

func TestNewEnvTemplateTagger(t *testing.T) {
	type args struct {
		t string
	}
	tests := []struct {
		name      string
		args      args
		shouldErr bool
	}{
		{
			name: "valid template",
			args: args{
				t: "{{.FOO}}",
			},
		},
		{
			name: "invalid template",
			args: args{
				t: "{{.FOO",
			},
			shouldErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnvTemplateTagger(tt.args.t)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}
