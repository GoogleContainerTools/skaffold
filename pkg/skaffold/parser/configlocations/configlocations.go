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

package configlocations

import (
	"context"
	"path"
	"reflect"
	"strconv"
	"strings"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
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

type YAMLInfos struct {
	yamlInfos               map[uintptr]map[string]YAMLInfo
	FieldsOverrodeByProfile map[string]YAMLOverrideInfo // map of schema path -> profile name -- ex: /artifacts/0/image -> "overwrite-artifacte-image-profile"
}

func (m *YAMLInfos) GetYamlInfosCopy() map[uintptr]map[string]YAMLInfo {
	yamlInfos := map[uintptr]map[string]YAMLInfo{}
	for ptr, mp := range m.yamlInfos {
		tmpmp := map[string]YAMLInfo{}
		for k, v := range mp {
			tmpmp[k] = YAMLInfo{
				RNode:      v.RNode.Copy(),
				SourceFile: v.SourceFile,
			}
		}
		yamlInfos[ptr] = tmpmp
	}
	return yamlInfos
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

func NewYAMLInfos() *YAMLInfos {
	return &YAMLInfos{
		yamlInfos: map[uintptr]map[string]YAMLInfo{},
	}
}

type YAMLOverrideInfo struct {
	ProfileName    string
	PatchIndex     int
	PatchOperation string
	PatchCopyFrom  string
}

// Parse parses a skaffold config entry collecting file location information for each schema config object
func Parse(sourceFile string, config *latestV2.SkaffoldConfig, fieldsOverrodeByProfile map[string]YAMLOverrideInfo) (*YAMLInfos, error) {
	yamlInfos, err := buildMapOfSchemaObjPointerToYAMLInfos(sourceFile, config, map[uintptr]map[string]YAMLInfo{}, fieldsOverrodeByProfile)
	return &YAMLInfos{
			yamlInfos:               yamlInfos,
			FieldsOverrodeByProfile: fieldsOverrodeByProfile,
		},
		err
}

// Locate gets the location for a skaffold schema struct pointer
func (m *YAMLInfos) Locate(obj interface{}) *Location {
	return m.locate(obj, "")
}

// Locate gets the location for a skaffold schema struct pointer
func (m *YAMLInfos) LocateElement(obj interface{}, idx int) *Location {
	return m.locate(obj, strconv.Itoa(idx))
}

// Locate gets the location for a skaffold schema struct pointer
func (m *YAMLInfos) LocateField(obj interface{}, fieldName string) *Location {
	return m.locate(obj, fieldName)
}

// Locate gets the location for a skaffold schema struct pointer
func (m *YAMLInfos) LocateByPointer(ptr uintptr) *Location {
	if m == nil {
		log.Entry(context.TODO()).Infof("YamlInfos is nil, unable to complete call to LocateByPointer for pointer: %d", ptr)
		return MissingLocation()
	}
	if _, ok := m.yamlInfos[ptr]; !ok {
		log.Entry(context.TODO()).Infof("no map entry found when attempting LocateByPointer for pointer: %d", ptr)
		return MissingLocation()
	}
	node, ok := m.yamlInfos[ptr][""]
	if !ok {
		log.Entry(context.TODO()).Infof("no map entry found when attempting LocateByPointer for pointer: %d", ptr)
		return MissingLocation()
	}
	// iterate over kyaml.RNode text to get endline and endcolumn information
	nodeText, err := node.RNode.String()
	if err != nil {
		return MissingLocation()
	}
	log.Entry(context.TODO()).Infof("map entry found when executing LocateByPointer for pointer: %d", ptr)
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

func (m *YAMLInfos) locate(obj interface{}, key string) *Location {
	if m == nil {
		log.Entry(context.TODO()).Infof("YamlInfos is nil, unable to complete call to locate with params: %v of type %T", obj, obj)
		return MissingLocation()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		log.Entry(context.TODO()).Infof("non pointer object passed to locate: %v of type %T", obj, obj)
		return MissingLocation()
	}
	if _, ok := m.yamlInfos[v.Pointer()]; !ok {
		log.Entry(context.TODO()).Infof("no map entry found when attempting locate for %v of type %T and pointer: %d", obj, obj, v.Pointer())
		return MissingLocation()
	}
	node, ok := m.yamlInfos[v.Pointer()][key]
	if !ok {
		log.Entry(context.TODO()).Infof("no map entry found when attempting locate for %v of type %T and pointer: %d", obj, obj, v.Pointer())
		return MissingLocation()
	}
	// iterate over kyaml.RNode text to get endline and endcolumn information
	nodeText, err := node.RNode.String()
	if err != nil {
		return MissingLocation()
	}
	log.Entry(context.TODO()).Infof("map entry found when executing locate for %v of type %T and pointer: %d", obj, obj, v.Pointer())
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

func buildMapOfSchemaObjPointerToYAMLInfos(sourceFile string, config *latestV2.SkaffoldConfig, yamlInfos map[uintptr]map[string]YAMLInfo,
	fieldsOverrodeByProfile map[string]YAMLOverrideInfo) (map[uintptr]map[string]YAMLInfo, error) {
	skaffoldConfigText, err := util.ReadConfiguration(sourceFile)
	if err != nil {
		return nil, sErrors.ConfigParsingError(err)
	}
	root, err := kyaml.Parse(string(skaffoldConfigText))
	if err != nil {
		return nil, err
	}
	// TODO(aaron-prindle) perhaps add some defensive logic to recover from panic when using reflection and instead return error?
	return generateObjPointerToYAMLNodeMap(sourceFile, reflect.ValueOf(config), reflect.ValueOf(nil), "", "", []string{},
		root, root, -1, fieldsOverrodeByProfile, map[interface{}]bool{}, yamlInfos, false)
}

// generateObjPointerToYAMLNodeMap recursively walks through a structs fields (taking into account profile and patch profile overrides)
// and collects the corresponding yaml node for each field
func generateObjPointerToYAMLNodeMap(sourceFile string, v reflect.Value, parentV reflect.Value, fieldName, yamlTag string, schemaPath []string,
	rootRNode *kyaml.RNode, rNode *kyaml.RNode, containerIdx int, fieldPathsOverrodeByProfiles map[string]YAMLOverrideInfo,
	visited map[interface{}]bool, yamlInfos map[uintptr]map[string]YAMLInfo, isPatchProfileElemOverride bool) (map[uintptr]map[string]YAMLInfo, error) {
	// TODO(aaron-prindle) need to verify if generateObjPointerToYAMLNodeMap adds entries for 'map' types, luckily the skaffold schema
	// only has map[string]string and they are leaf nodes as well which this should work fine for doing the recursion for the time being
	var err error

	// add current obj/field to schema path if criteria met
	switch {
	case containerIdx >= 0:
		schemaPath = append(schemaPath, strconv.Itoa(containerIdx))
	case yamlTag != "":
		schemaPath = append(schemaPath, yamlTag)
	}
	// check if current obj/field was overridden by a profile
	if yamlOverrideInfo, ok := fieldPathsOverrodeByProfiles["/"+path.Join(schemaPath...)]; ok {
		// reset yaml node path from root path to given profile path ("/" -> "/profile/name=profileName/etc...")
		rNode, err = rootRNode.Pipe(kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{yamlOverrideInfo.ProfileName}))
		if err != nil {
			return nil, err
		}
		switch {
		case yamlOverrideInfo.PatchIndex < 0: // this schema obj/field has a profile override (NOT a patch profile override)
			// moves parent node path from being rooted at default yaml '/' to being rooted at '/profile/name=profileName/...'
			for i := 0; i < len(schemaPath)-1; i++ {
				rNode, err = rNode.Pipe(kyaml.Lookup(schemaPath[i]))
				if err != nil {
					return nil, err
				}
			}
		default: // this schema obj/field has a patch profile override
			// NOTE: 'remove' patch operations are not included in fieldPathsOverrodeByProfiles as there
			//  is no work to be done on them (they were already removed from the schema)

			// TODO(aaron-prindle) verify UX makes sense to use the "FROM" copy node to get yaml information from
			if yamlOverrideInfo.PatchOperation == "copy" {
				fromPath := strings.Split(yamlOverrideInfo.PatchCopyFrom, "/")
				var kf kyaml.Filter
				for i := 0; i < len(fromPath)-1; i++ {
					if pathNum, err := strconv.Atoi(fromPath[i]); err == nil {
						// this path element is a number
						kf = kyaml.ElementIndexer{Index: pathNum}
					} else {
						// this path element isn't a number
						kf = kyaml.Lookup(fromPath[i])
					}
					rNode, err = rNode.Pipe(kf)
					if err != nil {
						return nil, err
					}
				}
			} else {
				rNode, err = rNode.Pipe(kyaml.Lookup("patches"), kyaml.ElementIndexer{Index: yamlOverrideInfo.PatchIndex})
				if err != nil {
					return nil, err
				}
				yamlTag = "value"
			}
			isPatchProfileElemOverride = true
		}
	}
	if rNode == nil {
		return yamlInfos, nil
	}

	// drill down through pointers and interfaces to get a value we can use
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.Kind() == reflect.Ptr {
			// Check for recursive data
			if visited[v.Interface()] {
				return yamlInfos, nil
			}
			visited[v.Interface()] = true
		}
		v = v.Elem()
	}

	if yamlTag != "" { // check that struct is not `yaml:",inline"`
		// traverse kyaml node tree to current obj/field location
		var kf kyaml.Filter
		switch {
		case rNode.YNode().Kind == kyaml.SequenceNode:
			kf = kyaml.ElementIndexer{Index: containerIdx}
		default:
			kf = kyaml.Lookup(yamlTag)
		}
		rNode, err = rNode.Pipe(kf)
		if err != nil {
			return nil, err
		}
		if rNode == nil {
			return yamlInfos, nil
		}

		// this case is so that the line #'s of primitive values can be "located" as they are not addressable but we can
		// map the parent address and put the child field in second map
		if parentV.CanAddr() {
			if _, ok := yamlInfos[parentV.Addr().Pointer()]; !ok {
				yamlInfos[parentV.Addr().Pointer()] = map[string]YAMLInfo{}
			}
			// add parent relationship entry to yaml info map
			if containerIdx >= 0 {
				yamlInfos[parentV.Addr().Pointer()][strconv.Itoa(containerIdx)] = YAMLInfo{
					RNode:      rNode,
					SourceFile: sourceFile,
				}
			} else {
				yamlInfos[parentV.Addr().Pointer()][fieldName] = YAMLInfo{
					RNode:      rNode,
					SourceFile: sourceFile,
				}
			}
		}
	}

	if v.CanAddr() {
		if _, ok := yamlInfos[v.Addr().Pointer()]; !ok {
			yamlInfos[v.Addr().Pointer()] = map[string]YAMLInfo{}
		}
		// add current node entry to yaml info map
		yamlInfos[v.Addr().Pointer()][""] = YAMLInfo{
			RNode:      rNode,
			SourceFile: sourceFile,
		}
	}

	switch v.Kind() {
	// TODO(aaron-prindle) add reflect.Map support here as well, currently no struct fields have nested struct in map field so ok for now
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			generateObjPointerToYAMLNodeMap(sourceFile, v.Index(i), v, fieldName+"["+strconv.Itoa(i)+"]", yamlTag+"["+strconv.Itoa(i)+"]", schemaPath,
				rootRNode, rNode, i, fieldPathsOverrodeByProfiles, visited, yamlInfos, isPatchProfileElemOverride)
		}
	case reflect.Struct:
		t := v.Type() // use type to get number and names of fields
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			// TODO(aaron-prindle) verify this value works for structs that are `yaml:",inline"`
			newYamlTag := field.Name
			if yamlTagToken := field.Tag.Get("yaml"); yamlTagToken != "" && yamlTagToken != "-" {
				// check for possible comma as in "...,omitempty"
				var commaIdx int
				if commaIdx = strings.Index(yamlTagToken, ","); commaIdx < 0 {
					commaIdx = len(yamlTagToken)
				}
				newYamlTag = yamlTagToken[:commaIdx]
			}
			generateObjPointerToYAMLNodeMap(sourceFile, v.Field(i), v, field.Name, newYamlTag, schemaPath, rootRNode, rNode, -1,
				fieldPathsOverrodeByProfiles, visited, yamlInfos, isPatchProfileElemOverride)
		}
	}
	return yamlInfos, nil
}
