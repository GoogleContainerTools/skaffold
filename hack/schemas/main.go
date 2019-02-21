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
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/pkg/errors"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

const defPrefix = "#/definitions/"

type Schema struct {
	*Definition
	Definitions *Definitions `json:"definitions,omitempty"`
}

type Definitions struct {
	keys   []string
	values map[string]*Definition
}

func (d *Definitions) Add(key string, value *Definition) {
	d.keys = append(d.keys, key)
	if d.values == nil {
		d.values = make(map[string]*Definition)
	}
	d.values[key] = value
}

func (d *Definitions) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("{")
	for i, k := range d.keys {
		if i != 0 {
			buf.WriteString(",")
		}
		// marshal key
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")

		// marshal value
		var val bytes.Buffer
		encoder := json.NewEncoder(&val)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(d.values[k]); err != nil {
			return nil, err
		}
		buf.Write(val.Bytes())
	}
	buf.WriteString("}")

	return buf.Bytes(), nil
}

type Definition struct {
	Ref                  string        `json:"$ref,omitempty"`
	Items                *Definition   `json:"items,omitempty"`
	Required             []string      `json:"required,omitempty"`
	Properties           *Definitions  `json:"properties,omitempty"`
	AdditionalProperties interface{}   `json:"additionalProperties,omitempty"`
	Type                 string        `json:"type,omitempty"`
	AnyOf                []*Definition `json:"anyOf,omitempty"`
	Description          string        `json:"description,omitempty"`
	Default              interface{}   `json:"default,omitempty"`
	Examples             []string      `json:"examples,omitempty"`
}

func main() {
	if _, err := generateSchemas(".", false); err != nil {
		panic(err)
	}
}

func generateSchemas(root string, dryRun bool) (bool, error) {
	same := true

	for i, version := range schema.SchemaVersions {
		apiVersion := strings.TrimPrefix(version.APIVersion, "skaffold/")
		folder := apiVersion
		if i == len(schema.SchemaVersions)-1 {
			folder = "latest"
		}

		input := fmt.Sprintf("%s/pkg/skaffold/schema/%s/config.go", root, folder)
		buf, err := generateSchema(input)
		if err != nil {
			return false, errors.Wrapf(err, "unable to generate schema for version %s", version.APIVersion)
		}

		output := fmt.Sprintf("%s/docs/content/en/schemas/%s.json", root, apiVersion)
		var current []byte

		if _, err := os.Stat(output); err == nil {
			var err error
			current, err = ioutil.ReadFile(output)
			if err != nil {
				return false, errors.Wrapf(err, "unable to read existing schema for version %s", version.APIVersion)
			}
		} else if !os.IsNotExist(err) {
			return false, errors.Wrapf(err, "unable to check that file exists %s", output)
		}

		if string(current) != string(buf) {
			same = false
		}

		if !dryRun {
			ioutil.WriteFile(output, buf, os.ModePerm)
		}
	}

	return same, nil
}

func yamlFieldName(field *ast.Field) string {
	tag := strings.Replace(field.Tag.Value, "`", "", -1)
	tags := reflect.StructTag(tag)
	yamlTag := tags.Get("yaml")

	return strings.Split(yamlTag, ",")[0]
}

func setTypeOrRef(def *Definition, typeName string) {
	switch typeName {
	case "string":
		def.Type = typeName
	case "bool":
		def.Type = "boolean"
	case "int", "int64":
		def.Type = "number"
	default:
		def.Ref = defPrefix + typeName
	}
}

func newDefinition(name string, t ast.Expr, comment string) *Definition {
	def := &Definition{}

	switch tt := t.(type) {
	case *ast.Ident:
		typeName := tt.Name
		setTypeOrRef(def, typeName)

		switch typeName {
		case "string":
			def.Default = "\"\""
		case "bool":
			def.Default = "false"
		case "int", "int64":
			def.Default = "0"
		}

	case *ast.StarExpr:
		typeName := tt.X.(*ast.Ident).Name
		setTypeOrRef(def, typeName)

	case *ast.ArrayType:
		def.Type = "array"
		def.Items = newDefinition("", tt.Elt, "")
		if def.Items.Ref == "" {
			def.Default = "[]"
		}

	case *ast.MapType:
		def.Type = "object"
		def.Default = "{}"
		def.AdditionalProperties = newDefinition("", tt.Value, "")

	case *ast.StructType:
		for _, field := range tt.Fields.List {
			yamlName := yamlFieldName(field)

			if strings.Contains(field.Tag.Value, "inline") {
				def.AnyOf = append(def.AnyOf, &Definition{
					Ref: defPrefix + field.Type.(*ast.Ident).Name,
				})
				continue
			}

			if yamlName == "" {
				continue
			}

			if strings.Contains(field.Tag.Value, "required") {
				def.Required = append(def.Required, yamlName)
			}

			if def.Properties == nil {
				def.Properties = &Definitions{}
			}

			def.Properties.Add(yamlName, newDefinition(field.Names[0].Name, field.Type, field.Doc.Text()))
			def.AdditionalProperties = false
		}
	}

	description := strings.TrimSpace(strings.Replace(comment, "\n", " ", -1))

	// Extract default value
	if m := regexp.MustCompile("(.*)Defaults to `(.*)`").FindStringSubmatch(description); m != nil {
		description = strings.TrimSpace(m[1])
		def.Default = m[2]
	}

	// Extract example
	if m := regexp.MustCompile("(.*)For example: `(.*)`").FindStringSubmatch(description); m != nil {
		description = strings.TrimSpace(m[1])
		def.Examples = []string{m[2]}
	}

	// Remove type prefix
	description = strings.TrimPrefix(description, name+" is the ")
	description = strings.TrimPrefix(description, name+" is ")
	description = strings.TrimPrefix(description, name+" are the ")
	description = strings.TrimPrefix(description, name+" are ")
	description = strings.TrimPrefix(description, name+" lists ")
	description = strings.TrimPrefix(description, name+" ")

	// Convert to HTML
	html := string(blackfriday.Run([]byte(description), blackfriday.WithNoExtensions()))
	html = strings.Replace(html, "<p>", "", -1)
	html = strings.Replace(html, "</p>", "", -1)
	def.Description = strings.TrimSpace(html)

	return def
}

func generateSchema(inputPath string) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	definitions := &Definitions{}

	for _, i := range node.Decls {
		declaration, ok := i.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range declaration.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			name := typeSpec.Name.Name
			definitions.Add(name, newDefinition(name, typeSpec.Type, declaration.Doc.Text()))
		}
	}

	// Inline anyOfs
	for _, v := range definitions.values {
		var options []*Definition

		for _, anyOf := range v.AnyOf {
			ref := strings.TrimPrefix(anyOf.Ref, defPrefix)
			referenced := definitions.values[ref]

			for _, key := range referenced.Properties.keys {
				choice := &Definitions{}
				choice.Add(key, referenced.Properties.values[key])

				options = append(options, &Definition{
					Properties: choice,
				})
			}
		}

		v.AnyOf = options
		v.AdditionalProperties = false
	}

	schema := Schema{
		Definition: &Definition{
			Type: "object",
			AnyOf: []*Definition{{
				Ref: defPrefix + "SkaffoldPipeline",
			}},
		},
		Definitions: definitions,
	}

	return toJSON(schema)
}

// Make sure HTML description are not encoded
func toJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
