// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolve

import (
	"errors"

	y "github.com/dprotaso/go-yit"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"
)

// MatchesSelector returns true if the Kubernetes object (represented as a
// yaml.Node) matches the selector. An error is returned if the yaml.Node is
// not an K8s object or list.
//
// If the document is a list, the yaml.Node will be mutated to only include
// items that match the selector.
func MatchesSelector(doc *yaml.Node, selector labels.Selector) (bool, error) {
	// ignore the document node
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}

	kind, err := docKind(doc)
	if err != nil {
		return false, err
	}
	if kind == "" {
		return false, nil
	}

	if kind == "List" {
		return listMatchesSelector(doc, selector)
	}

	return objMatchesSelector(doc, selector), nil
}

func docKind(doc *yaml.Node) (string, error) {
	// Null nodes will fail the check below, so simply ignore them.
	if doc.Tag == "!!null" {
		return "", nil
	}

	it := y.FromNode(doc).
		Filter(y.Intersect(
			y.WithKind(yaml.MappingNode),
			y.WithMapKeyValue(
				y.WithStringValue("apiVersion"),
				y.StringValue,
			),
		)).
		ValuesForMap(
			// Key Predicate
			y.WithStringValue("kind"),
			// Value Predicate
			y.StringValue,
		)

	node, ok := it()

	if !ok {
		return "", errors.New("yaml doesn't represent a k8s object")
	}

	return node.Value, nil
}

func objMatchesSelector(doc *yaml.Node, selector labels.Selector) bool {
	it := y.FromNode(doc).
		Filter(y.WithKind(yaml.MappingNode)).
		// Return the metadata map
		ValuesForMap(
			// Key Predicate
			y.WithStringValue("metadata"),
			// Value Predicate
			y.WithKind(yaml.MappingNode),
		).
		// Return the labels map
		ValuesForMap(
			// Key Predicate
			y.WithStringValue("labels"),
			// Value Predicate
			y.WithKind(yaml.MappingNode),
		)

	node, ok := it()

	// Object has no metadata.labels, verify matching against an empty set.
	if !ok {
		node = emptyMapNode
	}

	return selector.Matches(labelsNode{node})
}

func listMatchesSelector(doc *yaml.Node, selector labels.Selector) (bool, error) {
	it := y.FromNode(doc).ValuesForMap(
		// Key Predicate
		y.WithStringValue("items"),
		// Value Predicate
		y.WithKind(yaml.SequenceNode),
	)

	node, ok := it()

	// We don't have a k8s list
	if !ok {
		return false, errors.New("yaml is not a valid k8s list")
	}

	var matches []*yaml.Node
	for _, content := range node.Content {
		if _, err := docKind(content); err != nil {
			return false, err
		}

		if objMatchesSelector(content, selector) {
			matches = append(matches, content)
		}
	}

	node.Content = matches
	return len(matches) != 0, nil
}

var emptyMapNode = &yaml.Node{
	Kind: yaml.MappingNode,
	Tag:  "!!map",
}

type labelsNode struct {
	*yaml.Node
}

var _ labels.Labels = labelsNode{}

func (n labelsNode) Get(label string) (value string) {
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == label {
			return n.Content[i+1].Value
		}
	}
	return
}

func (n labelsNode) Has(label string) bool {
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == label {
			return true
		}
	}
	return false
}

func (n labelsNode) Lookup(label string) (value string, exists bool) {
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == label {
			return n.Content[i+1].Value, true
		}
	}
	return "", false
}
