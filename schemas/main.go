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
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/pkg/errors"
)

// Schema is the root schema.
// RFC draft-wright-json-schema-00, section 4.5
type Schema struct {
	*Definition
	Definitions Definitions `json:"definitions,omitempty"`
}

// Definitions hold schema definitions.
// http://json-schema.org/latest/json-schema-validation.html#rfc.section.5.26
// RFC draft-wright-json-schema-validation-00, section 5.26
type Definitions map[string]*Definition

// Definition type represents a JSON Schema object type.
type Definition struct {
	// RFC draft-wright-json-schema-00
	Version string `json:"$schema,omitempty"` // section 6.1
	Ref     string `json:"$ref,omitempty"`    // section 7
	// RFC draft-wright-json-schema-validation-00, section 5
	MultipleOf           int                    `json:"multipleOf,omitempty"`           // section 5.1
	Maximum              int                    `json:"maximum,omitempty"`              // section 5.2
	ExclusiveMaximum     bool                   `json:"exclusiveMaximum,omitempty"`     // section 5.3
	ExclusiveMinimum     bool                   `json:"exclusiveMinimum,omitempty"`     // section 5.5
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`          // section 5.12
	Minimum              int                    `json:"minimum,omitempty"`              // section 5.4
	MaxLength            int                    `json:"maxLength,omitempty"`            // section 5.6
	MinLength            int                    `json:"minLength,omitempty"`            // section 5.7
	Pattern              string                 `json:"pattern,omitempty"`              // section 5.8
	AdditionalItems      *Definition            `json:"additionalItems,omitempty"`      // section 5.9
	Items                *Definition            `json:"items,omitempty"`                // section 5.9
	MaxItems             int                    `json:"maxItems,omitempty"`             // section 5.10
	MinItems             int                    `json:"minItems,omitempty"`             // section 5.11
	MaxProperties        int                    `json:"maxProperties,omitempty"`        // section 5.13
	MinProperties        int                    `json:"minProperties,omitempty"`        // section 5.14
	Required             []string               `json:"required,omitempty"`             // section 5.15
	Properties           map[string]*Definition `json:"properties,omitempty"`           // section 5.16
	PatternProperties    map[string]*Definition `json:"patternProperties,omitempty"`    // section 5.17
	AdditionalProperties *Definition            `json:"additionalProperties,omitempty"` // section 5.18
	Dependencies         map[string]*Definition `json:"dependencies,omitempty"`         // section 5.19
	Enum                 []interface{}          `json:"enum,omitempty"`                 // section 5.20
	Type                 string                 `json:"type,omitempty"`                 // section 5.21
	AllOf                []*Definition          `json:"allOf,omitempty"`                // section 5.22
	AnyOf                []*Definition          `json:"anyOf,omitempty"`                // section 5.23
	OneOf                []*Definition          `json:"oneOf,omitempty"`                // section 5.24
	Not                  *Definition            `json:"not,omitempty"`                  // section 5.25
	Definitions          Definitions            `json:"definitions,omitempty"`          // section 5.26
	// RFC draft-wright-json-schema-validation-00, section 6, 7
	Title       string      `json:"title,omitempty"`       // section 6.1
	Description string      `json:"description,omitempty"` // section 6.1
	Default     interface{} `json:"default,omitempty"`     // section 6.2
	Format      string      `json:"format,omitempty"`      // section 7
	// RFC draft-wright-json-schema-hyperschema-00, section 4
	Media          *Definition `json:"media,omitempty"`          // section 4.3
	BinaryEncoding string      `json:"binaryEncoding,omitempty"` // section 4.3
}

const (
	defPrefix  = "#/definitions/"
	typeString = "string"
)

// Checks whether the typeName represents a simple json type
func isSimpleType(typeName string) bool {
	return typeName == typeString || typeName == "int" || typeName == "int64" || typeName == "bool"
}

// Converts the typeName simple type to json type
func jsonifyType(typeName string) string {
	switch typeName {
	case typeString:
		return typeString
	case "bool":
		return "boolean"
	case "int":
		return "number"
	case "int64":
		return "number"
	}
	panic("jsonifyType called with a complex type")
}

// Gets the type name of the array
func getTypeNameOfArray(arrayType *ast.ArrayType) string {
	switch at := arrayType.Elt.(type) {
	case *ast.Ident:
		return at.Name
	case *ast.StarExpr:
		identifier := at.X.(*ast.Ident)
		return identifier.Name
	}
	panic("undefined type")
}

// Removes a character by replacing it with a space
func removeChar(str string, removedStr string) string {
	return strings.Replace(str, removedStr, " ", -1)
}

// This is a hacky function that does the one job of
// extracting the tag values in the structs
// Example struct:
// type MyType struct {
//   MyField string `yaml:"myField,omitempty"`
// }
//
// From the above example struct, we need to extract
// and return this: ("myField", "omitempty")
func extractFromTag(tag *ast.BasicLit) (string, string) {
	tagValue := tag.Value
	if tagValue == "" {
		log.Panic("Tag value is empty")
	}

	// return yamlFieldValue, yamlOptionValue
	tagValue = removeChar(tagValue, "`")
	tagValue = removeChar(tagValue, `"`)
	tagValue = strings.TrimSpace(tagValue)

	var yamlTagContent string
	fmt.Sscanf(tagValue, `yaml: %s`, &yamlTagContent)

	if strings.Contains(yamlTagContent, ",") {
		splitContent := strings.Split(yamlTagContent, ",")
		return splitContent[0], splitContent[1]
	}
	return yamlTagContent, ""
}

// Gets the schema definition link of a resource
func getDefLink(resourceName string) string {
	return defPrefix + resourceName
}

// Parses array type and returns its corresponding
// schema definition.
func parseArrayType(arrayType *ast.ArrayType) *Definition {
	definition := new(Definition)
	typeNameOfArray := getTypeNameOfArray(arrayType)

	definition.Items = new(Definition)
	if isSimpleType(typeNameOfArray) {
		definition.Items.Type = jsonifyType(typeNameOfArray)
	} else {
		definition.Items.Ref = getDefLink(typeNameOfArray)
	}
	definition.Type = "array"

	return definition
}

// Merges the properties from the 'rhsDef' to the 'lhsDef'.
// Also transfers the description as well.
func mergeDefinitions(lhsDef *Definition, rhsDef *Definition) {
	// At this point, both defs will not have any 'AnyOf' defs.
	// 1. Add all the properties from rhsDef to lhsDef
	if lhsDef.Properties == nil {
		lhsDef.Properties = make(map[string]*Definition)
	}
	for propKey, propValue := range rhsDef.Properties {
		lhsDef.Properties[propKey] = propValue
	}
	// 2. Transfer the description
	if len(lhsDef.Description) == 0 {
		lhsDef.Description = rhsDef.Description
	}
}

// Gets the resource name from definitions url.
// Eg, returns 'TypeName' from '#/definitions/TypeName'
func getNameFromURL(url string) string {
	slice := strings.Split(url, "/")
	return slice[len(slice)-1]
}

// Recursively flattens "anyOf" tags. If there is cyclic
// dependency, execution is aborted.
func recursiveFlatten(schema *Schema, definition *Definition, defName string, visited *map[string]bool) *Definition {
	if len(definition.AllOf) == 0 {
		return definition
	}
	isAlreadyVisited := (*visited)[defName]
	if isAlreadyVisited {
		panic("Cycle detected in definitions")
	}
	(*visited)[defName] = true

	aggregatedDef := new(Definition)
	for _, allOfDef := range definition.AllOf {
		var newDef *Definition
		if allOfDef.Ref != "" {
			// If the definition has $ref url, fetch the referred resource
			// after flattening it.
			nameOfRef := getNameFromURL(allOfDef.Ref)
			newDef = recursiveFlatten(schema, schema.Definitions[nameOfRef], nameOfRef, visited)
		} else {
			newDef = allOfDef
		}
		mergeDefinitions(aggregatedDef, newDef)
	}

	delete(*visited, defName)
	return aggregatedDef
}

// Flattens the schema by inlining 'anyOf' tags.
func flattenSchema(schema *Schema) {
	for nameOfDef, def := range schema.Definitions {
		visited := make(map[string]bool)
		schema.Definitions[nameOfDef] = recursiveFlatten(schema, def, nameOfDef, &visited)
	}
}

// Parses a struct type and returns its corresponding
// schema definition.
func parseStructType(structType *ast.StructType, typeDescription string) *Definition {
	definition := &Definition{}
	definition.Description = typeDescription
	definition.Properties = make(map[string]*Definition)
	definition.Required = []string{}
	inlineDefinitions := []*Definition{}

	for _, field := range structType.Fields.List {
		property := new(Definition)
		yamlFieldName, option := extractFromTag(field.Tag)

		// If the 'inline' option is enabled, we need to merge
		// the type with its parent definition. We do it with
		// 'anyOf' json schema property.
		if option == "inline" {
			var typeName string
			switch ft := field.Type.(type) {
			case *ast.Ident:
				typeName = ft.String()
			case *ast.StarExpr:
				typeName = ft.X.(*ast.Ident).String()
			}
			inlinedDef := new(Definition)
			inlinedDef.Ref = getDefLink(typeName)
			inlineDefinitions = append(inlineDefinitions, inlinedDef)
			continue
		}
		// if 'omitempty' is not present, then the field is required
		if option != "omitempty" {
			definition.Required = append(definition.Required, yamlFieldName)
		}

		switch ft := field.Type.(type) {
		case *ast.Ident:
			typeName := ft.String()
			if isSimpleType(typeName) {
				property.Type = jsonifyType(typeName)
			} else {
				property.Ref = getDefLink(typeName)
			}
		case *ast.ArrayType:
			property = parseArrayType(ft)
		case *ast.MapType:
			switch fv := ft.Value.(type) {
			case *ast.Ident:
				property.AdditionalProperties = new(Definition)

				if isSimpleType(fv.Name) {
					property.AdditionalProperties.Type = fv.Name
				} else {
					property.AdditionalProperties.Ref = getDefLink(fv.Name)
				}
			case *ast.InterfaceType:
				// No op
			}
			property.Type = "object"
		case *ast.StarExpr:
			starType := ft.X.(*ast.Ident)
			typeName := starType.Name

			if isSimpleType(typeName) {
				property.Type = jsonifyType(typeName)
			} else {
				property.Ref = getDefLink(typeName)
			}
		}
		// Set the common properties here as the cases might
		// overwrite 'property' pointer.
		property.Description = field.Doc.Text()

		definition.Properties[yamlFieldName] = property
	}

	if len(inlineDefinitions) == 0 {
		// There are no inlined definitions
		return definition
	}

	// There are inlined definitions; we need to set
	// the "anyOf" property of a new parent node and attach
	// the inline definitions, along with the currently
	// parsed definition
	parentDefinition := new(Definition)

	if len(definition.Properties) != 0 {
		inlineDefinitions = append(inlineDefinitions, definition)
	}

	parentDefinition.AllOf = inlineDefinitions

	return parentDefinition
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
		buf, err := generateSchema(input, false)
		if err != nil {
			return false, errors.Wrapf(err, "unable to generate schema for version %s", version.APIVersion)
		}

		output := fmt.Sprintf("%s/schemas/%s.json", root, apiVersion)
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

func generateSchema(inputPath string, removeAllOfs bool) ([]byte, error) {
	// Open the input go file and parse the Abstract Syntax Tree
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	schema := Schema{
		Definition:  &Definition{},
		Definitions: make(map[string]*Definition)}
	schema.Type = "object"

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
			typeName := typeSpec.Name.Name
			typeDescription := declaration.Doc.Text()

			fmt.Println("Generating schema definition for type:", typeName)

			// Currently schema generation only takes Structs
			// and Array types into account.
			switch tt := typeSpec.Type.(type) {
			case *ast.ArrayType:
				schema.Definitions[typeName] = parseArrayType(tt)
			case *ast.StructType:
				schema.Definitions[typeName] = parseStructType(tt, typeDescription)
			}
		}
	}

	if removeAllOfs {
		fmt.Println("Flattening the schema by removing \"anyOf\" nodes")
		flattenSchema(&schema)
	}

	return json.MarshalIndent(schema, "", "  ")
}
