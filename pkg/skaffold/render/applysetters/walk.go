/*
Copyright 2023 The Skaffold Authors

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

package applysetters

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// visitor is implemented by structs which need to walk the configuration.
// visitor is provided to accept to walk configuration
type visitor interface {
	// visitScalar is called for each scalar field value on a resource
	// node is the scalar field value
	// path is the path to the field; path elements are separated by '.'
	visitScalar(node *yaml.RNode, path string) error

	// visitMapping is called for each Mapping field value on a resource
	// node is the mapping field value
	// path is the path to the field
	visitMapping(node *yaml.RNode, path string) error
}

// accept invokes the appropriate function on v for each field in object
func accept(v visitor, object *yaml.RNode) error {
	// get the OpenAPI for the type if it exists
	return acceptImpl(v, object, "")
}

// acceptImpl implements accept using recursion
func acceptImpl(v visitor, object *yaml.RNode, p string) error {
	switch object.YNode().Kind {
	case yaml.DocumentNode:
		// Traverse the child of the document
		return accept(v, yaml.NewRNode(object.YNode()))
	case yaml.MappingNode:
		if err := v.visitMapping(object, p); err != nil {
			return err
		}
		return object.VisitFields(func(node *yaml.MapNode) error {
			// Traverse each field value
			return acceptImpl(v, node.Value, p+"."+node.Key.YNode().Value)
		})
	case yaml.SequenceNode:
		return VisitElements(object, func(node *yaml.RNode, i int) error {
			// Traverse each list element
			return acceptImpl(v, node, p+fmt.Sprintf("[%d]", i))
		})
	case yaml.ScalarNode:
		// Visit the scalar field
		return v.visitScalar(object, p)
	}
	return nil
}

// VisitElements calls fn for each element in a SequenceNode.
// Returns an error for non-SequenceNodes
func VisitElements(rn *yaml.RNode, fn func(node *yaml.RNode, i int) error) error {
	elements, err := rn.Elements()
	if err != nil {
		return errors.Wrap(err)
	}
	for i := range elements {
		if err := fn(elements[i], i); err != nil {
			return errors.Wrap(err)
		}
	}
	return nil
}
