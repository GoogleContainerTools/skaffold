/*
Copyright 2024 The Skaffold Authors

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

package tags

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestApplyTemplates(t *testing.T) {
	type Inner struct {
		Name string `skaffold:"template"`
	}
	type testStruct struct {
		SimpleString  string             `skaffold:"template"`
		PtrString     *string            `skaffold:"template"`
		PtrPtrString  **string           `skaffold:"template"`
		SliceString   []string           `skaffold:"template"`
		ArrayString   [2]string          `skaffold:"template"`
		MapString     map[string]string  `skaffold:"template"`
		MapPtrString  map[string]*string `skaffold:"template"`
		IgnoredString string
		MapStruct     map[string]Inner
	}
	tests := []struct {
		name    string
		input   testStruct
		want    testStruct
		wantErr bool
		envs    map[string]string
	}{
		{
			name:    "Simple string",
			input:   testStruct{SimpleString: `Hello-{{.NAME}}`},
			want:    testStruct{SimpleString: `Hello-World`},
			wantErr: false,
			envs:    map[string]string{"NAME": "World"},
		},
		{
			name:    "Pointer to string",
			input:   testStruct{PtrString: util.Ptr(`Hello-{{.NAME}}`)},
			want:    testStruct{PtrString: util.Ptr(`Hello-World`)},
			wantErr: false,
			envs:    map[string]string{"NAME": "World"},
		},
		{
			name:    "Pointer to pointer to string",
			input:   testStruct{PtrPtrString: util.Ptr(util.Ptr(`Hello-{{.NAME}}`))},
			want:    testStruct{PtrPtrString: util.Ptr(util.Ptr(`Hello-World`))},
			wantErr: false,
			envs:    map[string]string{"NAME": "World"},
		},
		{
			name:    "Map of strings",
			input:   testStruct{MapString: map[string]string{"first": "first", "second": "{{.SECOND}}", "third": "{{.THIRD}}"}},
			want:    testStruct{MapString: map[string]string{"first": "first", "second": "second", "third": "third"}},
			wantErr: false,
			envs:    map[string]string{"SECOND": "second", "THIRD": "third"},
		},
		{
			name:  "Map of strings, keep the original template",
			input: testStruct{MapString: map[string]string{"first": "first", "second": "{{.SECOND}}", "third": "{{.THIRD}}"}},
			want:  testStruct{MapString: map[string]string{"first": "first", "second": "{{.SECOND}}", "third": "{{.THIRD}}"}},
			envs:  map[string]string{},
		},
		{
			name:    "Map of pointers to strings",
			input:   testStruct{MapPtrString: map[string]*string{"first": util.Ptr("first"), "second": util.Ptr("{{.SECOND}}"), "third": util.Ptr("{{.THIRD}}")}},
			want:    testStruct{MapPtrString: map[string]*string{"first": util.Ptr("first"), "second": util.Ptr("second"), "third": util.Ptr("third")}},
			wantErr: false,
			envs:    map[string]string{"SECOND": "second", "THIRD": "third"},
		},
		{
			name: "Array of strings",
			input: testStruct{
				ArrayString: [2]string{"{{ .ENV_VAR }}", "{{ .ENV_VAR }}"},
			},
			want: testStruct{
				ArrayString: [2]string{"replaced", "replaced"},
			},
			envs: map[string]string{"ENV_VAR": "replaced"},
		}, {
			name: "Ignored string",
			input: testStruct{
				IgnoredString: "{{ .ENV_VAR }}",
			},
			want: testStruct{
				IgnoredString: "{{ .ENV_VAR }}",
			},
			wantErr: false,
		},
		{
			name: "string not found - keep the original template",
			input: testStruct{
				SimpleString: "{{ .ENV_VAR }}",
			},
			want: testStruct{
				SimpleString: "{{ .ENV_VAR }}",
			},
		},
		{
			name: "invalid template",
			input: testStruct{
				SimpleString: "{{ .ENV_VAR ",
			},
			want: testStruct{
				SimpleString: "{{ .ENV_VAR ",
			},
			wantErr: true,
		},
		{
			name: "mapInner",
			input: testStruct{
				MapStruct: map[string]Inner{"aaa": {
					Name: "{{.NAME}}",
				}},
			},
			want: testStruct{
				MapStruct: map[string]Inner{"aaa": {
					Name: "World",
				}},
			},
			envs:    map[string]string{"NAME": "World"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testutil.Run(t, tt.name, func(t *testutil.T) {
			if tt.envs != nil {
				for k, v := range tt.envs {
					t.Setenv(k, v)
				}
			}

			if err := ApplyTemplates(&tt.input); (err != nil) != tt.wantErr {
				t.Errorf("applyTemplates() error = %v, wantErr %v", err, tt.wantErr)
			}
			t.CheckDeepEqual(tt.input, tt.want)
		})
	}
}

func Test_isSupportedType(t *testing.T) {
	tests := []struct {
		name string
		want bool
		v    reflect.Value
	}{
		{
			name: "Simple string",
			v:    reflect.ValueOf("test"),
			want: true,
		},
		{
			name: "Pointer to string",
			v:    reflect.ValueOf(util.Ptr("test")),
			want: true,
		},
		{
			name: "Slice of strings",
			v:    reflect.ValueOf([]string{"a", "b"}),
			want: true,
		},
		{
			name: "Array of strings",
			v:    reflect.ValueOf([2]string{"a", "b"}),
			want: true,
		},
		{
			name: "Map of strings",
			v:    reflect.ValueOf(map[string]string{"a": "b"}),
			want: true,
		},
		{
			name: "Slice of pointer to strings",
			v:    reflect.ValueOf([]*string{util.Ptr("test"), util.Ptr("test")}),
			want: true,
		},
		{
			name: "Unsupported type - int",
			v:    reflect.ValueOf(123),
			want: false,
		},
		{
			name: "Unsupported type - struct",
			v:    reflect.ValueOf(struct{}{}),
			want: false,
		},
		{
			name: "Empty slice",
			v:    reflect.ValueOf([]string{}),
			want: false,
		},
		{
			name: "Empty map",
			v:    reflect.ValueOf(map[string]string{}),
			want: false,
		},
	}
	for _, tt := range tests {
		testutil.Run(t, tt.name, func(t *testutil.T) {
			t.CheckDeepEqual(tt.want, isSupportedType(tt.v))
		})
	}
}

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		name    string
		v       reflect.Value
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Simple string",
			v:       reflect.ValueOf(util.Ptr("Hello, {{ .NAME }}!")),
			want:    "Hello, World!",
			wantErr: false,
		},
		{
			name:    "Pointer to string",
			v:       reflect.ValueOf(util.Ptr(util.Ptr("Hello, {{ .NAME }}!"))),
			want:    util.Ptr("Hello, World!"),
			wantErr: false,
		},
		{
			name:    "Slice of strings",
			v:       reflect.ValueOf([]string{"Hello, {{ .NAME }}!", "{{ .NAME }}, welcome!"}),
			want:    []string{"Hello, World!", "World, welcome!"},
			wantErr: false,
		},
		{
			name:    "Array of strings",
			v:       reflect.ValueOf(util.Ptr([2]string{"Hello, {{ .NAME }}!", "{{ .NAME }}, welcome!"})),
			want:    [2]string{"Hello, World!", "World, welcome!"},
			wantErr: false,
		},
		{
			name:    "Map of strings",
			v:       reflect.ValueOf(map[string]string{"greeting": "Hello, {{ .NAME }}!", "welcome": "{{ .NAME }}, welcome!"}),
			want:    map[string]string{"greeting": "Hello, World!", "welcome": "World, welcome!"},
			wantErr: false,
		},
		{
			name:    "Invalid template",
			v:       reflect.ValueOf(util.Ptr("{{ .INVALID ")),
			want:    "{{ .INVALID ",
			wantErr: true,
		},
		{
			name: "Map of pointers to strings",
			v: reflect.ValueOf(map[string]*string{
				"greeting": util.Ptr("Hello, {{ .NAME }}!"),
				"welcome":  util.Ptr("{{ .NAME }}, welcome!"),
			}),
			want: map[string]*string{
				"greeting": util.Ptr("Hello, World!"),
				"welcome":  util.Ptr("World, welcome!"),
			},
			wantErr: false,
		},
		{
			name:    "Slice of pointers to strings",
			v:       reflect.ValueOf([]*string{util.Ptr("Hello, {{ .NAME }}!"), util.Ptr("{{ .NAME }}, welcome!")}),
			want:    []*string{util.Ptr("Hello, World!"), util.Ptr("World, welcome!")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testutil.Run(t, tt.name, func(t *testutil.T) {
			// Set environment variable for templating
			t.Setenv("NAME", "World")
			err := expandTemplate(tt.v)
			t.CheckErrorAndDeepEqual(tt.wantErr, err, tt.want, reflect.Indirect(tt.v).Interface())
		})
	}
}

func TestContainTemplateTag(t *testing.T) {
	tests := []struct {
		name string
		sf   reflect.StructField
		want bool
	}{
		{
			name: "Tag with template",
			sf: reflect.StructField{
				Tag: reflect.StructTag(`skaffold:"template"`),
			},
			want: true,
		},
		{
			name: "Tag with template and other options",
			sf: reflect.StructField{
				Tag: reflect.StructTag(`skaffold:"template,option1,option2"`),
			},
			want: true,
		},
		{
			name: "Tag without template",
			sf: reflect.StructField{
				Tag: reflect.StructTag(`skaffold:"option1,option2"`),
			},
			want: false,
		},
		{
			name: "No skaffold tag",
			sf: reflect.StructField{
				Tag: reflect.StructTag(`json:"field"`),
			},
			want: false,
		},
		{
			name: "Empty tag",
			sf: reflect.StructField{
				Tag: reflect.StructTag(""),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		testutil.Run(t, tt.name, func(t *testutil.T) {
			got := containTemplateTag(tt.sf)
			t.CheckDeepEqual(tt.want, got)
		})
	}
}
