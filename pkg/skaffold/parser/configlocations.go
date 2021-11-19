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
package parser

import (
	"context"
	"reflect"
	"strings"
	"unicode"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type YAMLInfo struct {
	RNode      *kyaml.RNode
	SourceFile string
}

type Location struct {
	SourceFile  string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

type YAMLNodes struct {
	yamlNodes map[uintptr]*YAMLInfo
}

func MissingLocation() *Location {
	return &Location{
		SourceFile:  "",
		StartLine:   -1,
		StartColumn: -1,
		EndLine:     -1,
		EndColumn:   -1,
	}
}

func NewYAMLNodes() *YAMLNodes {
	return &YAMLNodes{
		yamlNodes: map[uintptr]*YAMLInfo{},
	}
}

// Parse parses a skaffold config entry collecting file location information for each schema config object
func Parse(c *SkaffoldConfigEntry) (*YAMLNodes, error) {
	yamlNodes, err := buildMapOfSchemaObjPointerToYAMLNodes(c, map[uintptr]*YAMLInfo{})
	return &YAMLNodes{yamlNodes}, err
}

// Locate gets the location for a skaffold schema struct pointer
func (m *YAMLNodes) Locate(obj interface{}) *Location {
	kind := reflect.ValueOf(obj).Kind()
	if kind != reflect.Ptr {
		log.Entry(context.TODO()).Infof("non pointer object passed to Locate: %v of type %T", obj, obj)
		return MissingLocation()
	}

	node := m.yamlNodes[reflect.ValueOf(obj).Pointer()]
	if node == nil {
		log.Entry(context.TODO()).Infof("no map entry found when attempting Locate for %v of type %T", obj, obj)
		return MissingLocation()
	}

	// iterate over kyaml.RNode to get endline and endcolumn
	nodeText, err := node.RNode.String()
	if err != nil {
		return MissingLocation()
	}
	lines, cols := getLinesAndColsOfString(nodeText)

	// TODO(aaron-prindle) all line & col values seem 1 greater than expected in actual use, will need to check to see how it works with IDE
	return &Location{
		SourceFile:  node.SourceFile,
		StartLine:   node.RNode.Document().Line,
		StartColumn: node.RNode.Document().Column,
		EndLine:     node.RNode.Document().Line + lines,
		EndColumn:   cols,
	}
}

func getLinesAndColsOfString(str string) (int, int) {
	line := 0
	col := 0
	for i := range str {
		col++
		if str[i] == '\n' {
			line++
			col = 0
		}
	}
	return line, col
}

func buildMapOfSchemaObjPointerToYAMLNodes(c *SkaffoldConfigEntry, yamlNodes map[uintptr]*YAMLInfo) (map[uintptr]*YAMLInfo, error) {
	skaffoldConfigText, err := util.ReadConfiguration(c.SourceFile)
	if err != nil {
		return nil, sErrors.ConfigParsingError(err)
	}
	root, err := kyaml.Parse(string(skaffoldConfigText))
	if err != nil {
		return nil, err
	}
	// TODO(aaron-prindle) perhaps add some defensive logic to recover from panic when using reflection and instead return error?
	return generateObjPointerToYAMLNodeMap(c.SourceFile, reflect.ValueOf(c), "", root, -1, map[interface{}]bool{}, yamlNodes)
}

// generateObjPointerToYAMLNodeMap recursively walks through a structs fields and collects the yaml nodes for each related field
func generateObjPointerToYAMLNodeMap(sourceFile string, v reflect.Value, yamlField string, parentRNode *kyaml.RNode,
	idx int, visited map[interface{}]bool, yamlNodes map[uintptr]*YAMLInfo) (map[uintptr]*YAMLInfo, error) {
	// TODO(aaron-prindle) don't believe this will work properly for 'map' types, luckily the skaffold schema only has map[string]string which
	// and they are leaf nodes as well.
	var err error
	var first rune
	for _, c := range yamlField {
		first = c
		break
	}
	if yamlField != "" && unicode.IsLower(first) { // lowercase check identifies fields as yaml fields in skaffold's schema spec (vs default values like SourceFile, etc.)
		switch {
		case parentRNode == nil:
			return yamlNodes, nil
		case parentRNode.YNode().Kind == kyaml.SequenceNode:
			elems, err := parentRNode.Elements()
			if err != nil {
				return yamlNodes, err
			}
			parentRNode = elems[idx]
		default:
			parentRNode, err = parentRNode.Pipe(kyaml.Lookup(yamlField))
			if err != nil {
				return yamlNodes, err
			}
		}
		yamlNodes[v.Addr().Pointer()] = &YAMLInfo{
			RNode:      parentRNode,
			SourceFile: sourceFile,
		}
	}

	// Drill down through pointers and interfaces to get a value we can print.
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.Kind() == reflect.Ptr {
			// Check for recursive data
			if visited[v.Interface()] {
				return yamlNodes, nil
			}
			visited[v.Interface()] = true
		}
		v = v.Elem()
	}

	switch v.Kind() {
	// TODO(aaron-prindle) add reflect.Map support here as well, currently no structs have nested struct in map field so ok for now
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			generateObjPointerToYAMLNodeMap(sourceFile, v.Index(i), "", parentRNode, i, visited, yamlNodes)
		}
	case reflect.Struct:
		t := v.Type() // use type to get number and names of fields
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldName := field.Name
			if yamlTag := field.Tag.Get("yaml"); yamlTag != "" && yamlTag != "-" {
				// check for possible comma as in "...,omitempty"
				var commaIdx int
				if commaIdx = strings.Index(yamlTag, ","); commaIdx < 0 {
					commaIdx = len(yamlTag)
				}
				fieldName = yamlTag[:commaIdx]
			}
			generateObjPointerToYAMLNodeMap(sourceFile, v.Field(i), fieldName, parentRNode, idx, visited, yamlNodes)
		}
	}
	return yamlNodes, nil
}
