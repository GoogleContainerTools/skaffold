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
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/xeipuuv/gojsonschema"
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
	tcs := []struct {
		name string
	}{
		{name: "inline"},
		{name: "inline-anyof"},
		{name: "inline-hybrid"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			input := fmt.Sprintf("./testdata/%s/input.go", tc.name)
			expectedOutput := fmt.Sprintf("./testdata/%s/output.json", tc.name)

			generator := schemaGenerator{
				strict: false,
			}

			actual, err := generator.Apply(input)
			testutil.CheckError(t, false, err)

			var expected []byte
			if _, err := os.Stat(expectedOutput); err == nil {
				var err error
				expected, err = ioutil.ReadFile(expectedOutput)
				testutil.CheckError(t, false, err)
			}

			if diff := cmp.Diff(string(actual), string(expected)); diff != "" {
				t.Errorf("%T differ (-got, +want): %s\n actual:\n%s", string(expected), diff, string(actual))
				return
			}
		})
	}
}

func TestValidateJsonSchemas(t *testing.T) {
	root := "../.."

	for _, v := range schema.SchemaVersions {
		apiVersion := strings.TrimPrefix(v.APIVersion, "skaffold/")
		t.Run(apiVersion, func(t *testing.T) {
			yamlMapConfig, err := toMapWithYamlKeys(v.Factory())
			if err != nil {
				t.Fatalf("cannot convert to config with yaml names")
			}
			document := gojsonschema.NewGoLoader(yamlMapConfig)

			rawSchema := loadSchemaForAPIVersion(t, root, apiVersion)
			jsonSchema := gojsonschema.NewStringLoader(rawSchema)

			result, err := gojsonschema.Validate(jsonSchema, document)
			if err != nil {
				t.Fatalf("schema validation failed: %s", err)
			}

			if !result.Valid() {
				for _, desc := range result.Errors() {
					t.Error(desc)
				}
			}
		})
	}
}

func loadSchemaForAPIVersion(t *testing.T, root, apiVersion string) string {
	t.Helper()
	schemaPath := fmt.Sprintf("%s/docs/content/en/schemas/%s.json", root, apiVersion)
	if _, err := os.Stat(schemaPath); err == nil {
		if content, err := ioutil.ReadFile(schemaPath); err != nil {
			t.Errorf("unable to read existing schema for version %s", apiVersion)
		} else {
			return string(content)
		}
	} else if os.IsNotExist(err) {
		t.Skip("could not find schema at", schemaPath)
	}
	return ""
}

// toMapWithYamlKeys converts any given interface which can be yaml-marshalled to
// a go object of type map[string]interface{}, where the object keys are the yaml names
func toMapWithYamlKeys(in interface{}) (interface{}, error) {
	// get yaml with the correct field names
	yamlBytes, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}

	// convert yamlBytes to document of type map[interface{}]interface{}
	var document interface{}
	r := bytes.NewReader(yamlBytes)
	decoder := yaml.NewDecoder(r)
	err = decoder.Decode(&document)
	if err != nil {
		return nil, err
	}

	// convert document to type map[string]interface{}
	return fixMapKeys(document), nil
}

// fixMapKeys converts an object of type map[interface{}]interface{} to map[string]interface{}.
func fixMapKeys(in interface{}) interface{} {
	switch val := in.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{})
		for k, v := range val {
			switch ks := k.(type) {
			case string:
				out[ks] = fixMapKeys(v)
			default:
				out[fmt.Sprint(k)] = fixMapKeys(v)
			}
		}
		return out

	case []interface{}:
		for i, v := range val {
			val[i] = fixMapKeys(v)
		}
	}
	return in
}
