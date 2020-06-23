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
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	blackfriday "github.com/russross/blackfriday/v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
)

const (
	version7  = "http://json-schema-org/draft-07/schema#"
	defPrefix = "#/definitions/"
)

var (
	regexpDefaults = regexp.MustCompile("(.*)Defaults to `(.*)`")
	regexpExample  = regexp.MustCompile("(.*)For example: `(.*)`")
	pTags          = regexp.MustCompile("(<p>)|(</p>)")

	// patterns for enum-type values
	enumValuePattern     = "^[ \t]*`(?P<name>[^`]+)`([ \t]*\\(default\\))?: .*$"
	regexpEnumDefinition = regexp.MustCompile("(?m).*Valid [a-z]+ are((\\n" + enumValuePattern + ")*)")
	regexpEnumValues     = regexp.MustCompile("(?m)" + enumValuePattern)
)

type schemaGenerator struct {
	strict bool
}

type Schema struct {
	*Definition
	Version     string                 `json:"$schema,omitempty"`
	Definitions map[string]*Definition `json:"definitions,omitempty"`
}

type Definition struct {
	Ref                  string                 `json:"$ref,omitempty"`
	Items                *Definition            `json:"items,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           map[string]*Definition `json:"properties,omitempty"`
	PreferredOrder       []string               `json:"preferredOrder,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	AnyOf                []*Definition          `json:"anyOf,omitempty"`
	Description          string                 `json:"description,omitempty"`
	HTMLDescription      string                 `json:"x-intellij-html-description,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Examples             []string               `json:"examples,omitempty"`
	Enum                 []string               `json:"enum,omitempty"`

	inlines []*Definition
	tags    string
}

func main() {
	if _, err := generateSchemas(".", false); err != nil {
		panic(err)
	}
}

type sameErr struct {
	same bool
	err  error
}

func generateSchemas(root string, dryRun bool) (bool, error) {
	var results [](chan sameErr)
	for range schema.SchemaVersions {
		results = append(results, make(chan sameErr, 1))
	}

	var wg sync.WaitGroup
	for i, version := range schema.SchemaVersions {
		wg.Add(1)
		go func(i int, version schema.Version) {
			same, err := generateSchema(root, dryRun, version)
			results[i] <- sameErr{
				same: same,
				err:  err,
			}
			wg.Done()
		}(i, version)
	}
	wg.Wait()

	same := true
	for i := range schema.SchemaVersions {
		result := <-results[i]
		if result.err != nil {
			return false, result.err
		}

		same = same && result.same
	}

	return same, nil
}

func generateSchema(root string, dryRun bool, version schema.Version) (bool, error) {
	apiVersion := strings.TrimPrefix(version.APIVersion, "skaffold/")

	folder := apiVersion
	strict := false
	if version.APIVersion == schema.SchemaVersions[len(schema.SchemaVersions)-1].APIVersion {
		folder = "latest"
		strict = true
	}

	input := filepath.Join(root, "pkg", "skaffold", "schema", folder, "config.go")
	output := filepath.Join(root, "docs", "content", "en", "schemas", apiVersion+".json")

	generator := schemaGenerator{
		strict: strict,
	}

	buf, err := generator.Apply(input)
	if err != nil {
		return false, fmt.Errorf("unable to generate schema for version %q: %w", version.APIVersion, err)
	}

	var current []byte
	if _, err := os.Stat(output); err == nil {
		var err error
		current, err = ioutil.ReadFile(output)
		if err != nil {
			return false, fmt.Errorf("unable to read existing schema for version %q: %w", version.APIVersion, err)
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("unable to check that file exists %q: %w", output, err)
	}

	current = bytes.Replace(current, []byte("\r\n"), []byte("\n"), -1)

	if !dryRun {
		if err := ioutil.WriteFile(output, buf, os.ModePerm); err != nil {
			return false, fmt.Errorf("unable to write schema %q: %w", output, err)
		}
	}

	same := string(current) == string(buf)
	return same, nil
}

func yamlFieldName(field *ast.Field) string {
	tag := strings.Replace(field.Tag.Value, "`", "", -1)
	tags := reflect.StructTag(tag)
	yamlTag := tags.Get("yaml")

	return strings.Split(yamlTag, ",")[0]
}

//nolint:golint,goconst
func setTypeOrRef(def *Definition, typeName string) {
	switch typeName {
	// Special case for ResourceType that is an alias of string.
	// Fixes #3623
	case "string", "ResourceType":
		def.Type = "string"
	case "bool":
		def.Type = "boolean"
	case "int", "int64", "int32":
		def.Type = "integer"
	default:
		def.Ref = defPrefix + typeName
	}
}

func (g *schemaGenerator) newDefinition(name string, t ast.Expr, comment string, tags string) *Definition {
	def := &Definition{
		tags: tags,
	}

	switch tt := t.(type) {
	case *ast.Ident:
		typeName := tt.Name
		setTypeOrRef(def, typeName)

		switch typeName {
		case "string":
			// def.Default = "\"\""
		case "bool":
			def.Default = "false"
		case "int", "int64":
			// def.Default = "0"
		}

	case *ast.StarExpr:
		if ident, ok := tt.X.(*ast.Ident); ok {
			typeName := ident.Name
			setTypeOrRef(def, typeName)
		}

	case *ast.ArrayType:
		def.Type = "array"
		def.Items = g.newDefinition("", tt.Elt, "", "")
		if def.Items.Ref == "" {
			def.Default = "[]"
		}

	case *ast.MapType:
		def.Type = "object"
		def.Default = "{}"
		def.AdditionalProperties = g.newDefinition("", tt.Value, "", "")

	case *ast.StructType:
		for _, field := range tt.Fields.List {
			yamlName := yamlFieldName(field)

			if strings.Contains(field.Tag.Value, "inline") {
				def.PreferredOrder = append(def.PreferredOrder, "<inline>")
				def.inlines = append(def.inlines, &Definition{
					Ref: defPrefix + field.Type.(*ast.Ident).Name,
				})
				continue
			}

			if yamlName == "" || yamlName == "-" {
				continue
			}

			if strings.Contains(field.Tag.Value, "required") {
				def.Required = append(def.Required, yamlName)
			}

			if def.Properties == nil {
				def.Properties = make(map[string]*Definition)
			}

			def.PreferredOrder = append(def.PreferredOrder, yamlName)
			def.Properties[yamlName] = g.newDefinition(field.Names[0].Name, field.Type, field.Doc.Text(), field.Tag.Value)
			def.AdditionalProperties = false
		}
	}

	if g.strict && name != "" {
		if !strings.HasPrefix(comment, name+" ") {
			panic(fmt.Sprintf("comment should start with field name on field %s", name))
		}
	}

	// process enums before stripping out newlines
	if m := regexpEnumDefinition.FindStringSubmatch(comment); m != nil {
		enums := make([]string, 0)
		if n := regexpEnumValues.FindAllStringSubmatch(m[1], -1); n != nil {
			for _, matches := range n {
				enums = append(enums, matches[1])
			}
			def.Enum = enums
		}
	}

	description := strings.TrimSpace(strings.Replace(comment, "\n", " ", -1))

	// Extract default value
	if m := regexpDefaults.FindStringSubmatch(description); m != nil {
		description = strings.TrimSpace(m[1])
		def.Default = m[2]
	}

	// Extract example
	if m := regexpExample.FindStringSubmatch(description); m != nil {
		description = strings.TrimSpace(m[1])
		def.Examples = []string{m[2]}
	}

	// Remove type prefix
	description = regexp.MustCompile("^"+name+" (\\*.*\\* )?((is (the )?)|(are (the )?)|(lists ))?").ReplaceAllString(description, "$1")

	if g.strict && name != "" {
		if description == "" {
			panic(fmt.Sprintf("no description on field %s", name))
		}
		if !strings.HasSuffix(description, ".") {
			panic(fmt.Sprintf("description should end with a dot on field %s", name))
		}
	}
	def.Description = description

	// Convert to HTML
	html := string(blackfriday.Run([]byte(description), blackfriday.WithNoExtensions()))
	def.HTMLDescription = strings.TrimSpace(pTags.ReplaceAllString(html, ""))

	return def
}

func isOneOf(definition *Definition) bool {
	return len(definition.Properties) > 0 &&
		strings.Contains(definition.Properties[definition.PreferredOrder[0]].tags, "oneOf=")
}

func (g *schemaGenerator) Apply(inputPath string) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var preferredOrder []string
	definitions := make(map[string]*Definition)

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
			preferredOrder = append(preferredOrder, name)
			definitions[name] = g.newDefinition(name, typeSpec.Type, declaration.Doc.Text(), "")
		}
	}

	var inlines []string

	for _, k := range preferredOrder {
		def := definitions[k]
		if len(def.inlines) == 0 {
			continue
		}

		for _, inlineStruct := range def.inlines {
			ref := strings.TrimPrefix(inlineStruct.Ref, defPrefix)
			inlines = append(inlines, ref)
		}

		// First, inline definitions without `oneOf`
		inlineIndex := 0
		var defPreferredOrder []string
		for _, k := range def.PreferredOrder {
			if k != "<inline>" {
				defPreferredOrder = append(defPreferredOrder, k)
				continue
			}

			inlineStruct := def.inlines[inlineIndex]
			inlineIndex++

			ref := strings.TrimPrefix(inlineStruct.Ref, defPrefix)
			inlineStructRef := definitions[ref]
			if isOneOf(inlineStructRef) {
				continue
			}

			if def.Properties == nil {
				def.Properties = make(map[string]*Definition, len(inlineStructRef.Properties))
			}
			for k, v := range inlineStructRef.Properties {
				def.Properties[k] = v
			}
			defPreferredOrder = append(defPreferredOrder, inlineStructRef.PreferredOrder...)
			def.Required = append(def.Required, inlineStructRef.Required...)
		}
		def.PreferredOrder = defPreferredOrder

		// Then add options for `oneOf` definitions
		var options []*Definition
		for _, inlineStruct := range def.inlines {
			ref := strings.TrimPrefix(inlineStruct.Ref, defPrefix)
			inlineStructRef := definitions[ref]
			if !isOneOf(inlineStructRef) {
				continue
			}

			for _, key := range inlineStructRef.PreferredOrder {
				var preferredOrder []string
				choice := make(map[string]*Definition)

				if len(def.Properties) > 0 {
					for _, pkey := range def.PreferredOrder {
						preferredOrder = append(preferredOrder, pkey)
						choice[pkey] = def.Properties[pkey]
					}
				}

				preferredOrder = append(preferredOrder, key)
				choice[key] = inlineStructRef.Properties[key]

				options = append(options, &Definition{
					Properties:           choice,
					PreferredOrder:       preferredOrder,
					AdditionalProperties: false,
				})
			}
		}

		if len(options) == 0 {
			continue
		}

		options = append([]*Definition{{
			Properties:           def.Properties,
			PreferredOrder:       def.PreferredOrder,
			AdditionalProperties: false,
		}}, options...)

		def.Properties = nil
		def.PreferredOrder = nil
		def.AdditionalProperties = nil
		def.AnyOf = options
	}

	for _, ref := range inlines {
		delete(definitions, ref)
	}

	schema := Schema{
		Version: version7,
		Definition: &Definition{
			Type: "object",
			AnyOf: []*Definition{{
				Ref: defPrefix + preferredOrder[0],
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
