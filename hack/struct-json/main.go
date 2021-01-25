/*
Copyright 2021 The Skaffold Authors

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
	"regexp"
	"strings"

	blackfriday "github.com/russross/blackfriday/v2"
)

var camelSepRegex = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type Doc struct {
	*Definition
	Definitions map[string]*Definition `json:"definitions,omitempty"`
}

type Definition struct {
	Items           *Definition            `json:"items,omitempty"`
	Properties      map[string]*Definition `json:"properties,omitempty"`
	Description     string                 `json:"description,omitempty"`
	HTMLDescription string                 `json:"x-intellij-html-description,omitempty"`
}

func main() {
	if len(os.Args) < 4 {
		panic(fmt.Errorf("not enough arguments"))
	}
	if err := generateJSON(".", os.Args[2], os.Args[3], false); err != nil {
		panic(err)
	}
}

func generateJSON(root, input, output string, dryRun bool) error {
	buf, err := generate(filepath.Join(root, input))
	if err != nil {
		return fmt.Errorf("unable to generate json with comments for %s %v", input, err)
	}

	if !dryRun {
		if err := ioutil.WriteFile(output, buf, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write json %q: %w", output, err)
		}
	}

	return nil
}

func newDefinition(name string, t ast.Expr, comment string) *Definition {
	def := &Definition{}
	tt, ok := t.(*ast.StructType)
	if ok {
		for _, field := range tt.Fields.List {
			name := string(camelSepRegex.ReplaceAll([]byte(field.Names[0].Name), []byte("$1 $2")))
			if def.Properties == nil {
				def.Properties = make(map[string]*Definition)
			}

			def.Properties[name] = newDefinition(name, field.Type, field.Doc.Text())
		}
	}

	ogName := strings.ReplaceAll(name, " ", "")
	if name != "" {
		if comment == "" {
			panic(fmt.Sprintf("field %q needs comment (all public fields require comments)", name))
		}
		if !strings.HasPrefix(comment, ogName+" ") {
			panic(fmt.Sprintf("comment %q should start with field name on field %s", comment, name))
		}
	}

	description := strings.TrimSpace(strings.Replace(comment, "\n", " ", -1))
	// Remove type prefix
	description = regexp.MustCompile("^"+ogName+" (\\*.*\\* )?((is (the )?)|(are (the )?)|(lists ))?").ReplaceAllString(description, "$1")

	if name != "" {
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
	def.HTMLDescription = strings.TrimSpace(html)

	return def
}

func generate(inputPath string) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

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
			definitions[name] = newDefinition(name, typeSpec.Type, declaration.Doc.Text())
		}
	}

	doc := Doc{
		Definitions: definitions,
	}

	return toJSON(doc)
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
