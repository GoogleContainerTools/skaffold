/*
Copyright 2026 The Skaffold Authors

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

package schema

import (
	"fmt"
	"reflect"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

var yamlMarshalerType = reflect.TypeOf((*yamlv3.Marshaler)(nil)).Elem()

// marshalForPatching marshals a config the same way as yaml.Marshal, except that
// maps and slices which are empty but non-nil are kept instead of being dropped by
// `omitempty`. A user who writes `buildArgs: {}` or `releases: []` should be able to
// `add` into that collection from a profile patch, which requires the (empty) parent
// to still be present in the document the patch is applied to.
func marshalForPatching(in interface{}) ([]byte, error) {
	var node yamlv3.Node
	if err := node.Encode(in); err != nil {
		return nil, err
	}
	restoreEmptyCollections(reflect.ValueOf(in), &node)
	return yaml.Marshal(&node)
}

// restoreEmptyCollections walks v alongside its encoded node and adds back the
// empty, non-nil maps and slices that were omitted. It reports whether it added
// anything to node.
func restoreEmptyCollections(v reflect.Value, node *yamlv3.Node) bool {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if node.Kind == yamlv3.DocumentNode && len(node.Content) == 1 {
		return restoreEmptyCollections(v, node.Content[0])
	}
	// Types with custom marshaling control their own encoding.
	if v.Type().Implements(yamlMarshalerType) || reflect.PointerTo(v.Type()).Implements(yamlMarshalerType) {
		return false
	}

	switch {
	case v.Kind() == reflect.Struct && node.Kind == yamlv3.MappingNode:
		return restoreStructFields(v, node)
	case v.Kind() == reflect.Slice && node.Kind == yamlv3.SequenceNode && v.Len() == len(node.Content):
		added := false
		for i := 0; i < v.Len(); i++ {
			added = restoreEmptyCollections(v.Index(i), node.Content[i]) || added
		}
		return added
	case v.Kind() == reflect.Map && node.Kind == yamlv3.MappingNode:
		added := false
		for i := 0; i+1 < len(node.Content); i += 2 {
			for _, key := range v.MapKeys() {
				if fmt.Sprint(key.Interface()) == node.Content[i].Value {
					added = restoreEmptyCollections(v.MapIndex(key), node.Content[i+1]) || added
					break
				}
			}
		}
		return added
	}
	return false
}

func restoreStructFields(v reflect.Value, node *yamlv3.Node) bool {
	added := false
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		fv := v.Field(i)
		if strings.Contains(opts, "inline") {
			added = restoreEmptyCollections(fv, node) || added
			continue
		}

		if valueNode := mappingValue(node, name); valueNode != nil {
			added = restoreEmptyCollections(fv, valueNode) || added
			continue
		}

		// The field was omitted. Put it back if it is an empty but non-nil
		// collection, or a struct that contains one.
		for fv.Kind() == reflect.Pointer && !fv.IsNil() {
			fv = fv.Elem()
		}
		var valueNode *yamlv3.Node
		switch fv.Kind() {
		case reflect.Map:
			if !fv.IsNil() && fv.Len() == 0 {
				valueNode = &yamlv3.Node{Kind: yamlv3.MappingNode, Tag: "!!map", Style: yamlv3.FlowStyle}
			}
		case reflect.Slice:
			if !fv.IsNil() && fv.Len() == 0 {
				valueNode = &yamlv3.Node{Kind: yamlv3.SequenceNode, Tag: "!!seq", Style: yamlv3.FlowStyle}
			}
		case reflect.Struct:
			structNode := &yamlv3.Node{Kind: yamlv3.MappingNode, Tag: "!!map"}
			if restoreEmptyCollections(fv, structNode) {
				valueNode = structNode
			}
		}
		if valueNode != nil {
			// An empty parent is encoded in flow style (`{}`); switch it to block style now that it has content.
			node.Style &^= yamlv3.FlowStyle
			node.Content = append(node.Content, &yamlv3.Node{Kind: yamlv3.ScalarNode, Tag: "!!str", Value: name}, valueNode)
			added = true
		}
	}
	return added
}

func mappingValue(node *yamlv3.Node, key string) *yamlv3.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
