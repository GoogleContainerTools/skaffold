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

package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/xeipuuv/gojsonschema"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSchemas(t *testing.T) {
	same, err := generateSchemas("../..", true)
	if err != nil {
		t.Fatalf("unable to check json schemas: %v", err)
	}

	if !same {
		t.Fatal("json schema files are not up to date. Please run `make generate-schemas` and commit the changes.")
	}
}

func TestGenerators(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "inline"},
		{name: "inline-anyof"},
		{name: "inline-hybrid"},
		{name: "integer"},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			input := filepath.Join("testdata", test.name, "input.go")
			expectedOutput := filepath.Join("testdata", test.name, "output.json")

			generator := schemaGenerator{
				strict: false,
			}

			actual, err := generator.Apply(input)
			t.CheckNoError(err)

			expected, err := ioutil.ReadFile(expectedOutput)
			t.CheckNoError(err)

			expected = bytes.Replace(expected, []byte("\r\n"), []byte("\n"), -1)

			schemaLoader := gojsonschema.NewBytesLoader(actual)
			_, err = gojsonschema.NewSchema(schemaLoader)
			t.CheckNoError(err)

			if diff := cmp.Diff(string(actual), string(expected)); diff != "" {
				t.Errorf("%T differ (-got, +want): %s\n actual:\n%s", string(expected), diff, string(actual))
				return
			}
		})
	}
}
