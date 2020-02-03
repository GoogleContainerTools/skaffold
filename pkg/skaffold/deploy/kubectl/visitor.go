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

package kubectl

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// FieldVisitor represents the aggregation/transformation that should be performed on each traversed field.
type FieldVisitor interface {
	// Visit is called for each transformable key contained in the object and may apply transformations/aggregations on it.
	// It should return true to allow recursive traversal or false when the entry was transformed.
	Visit(object map[interface{}]interface{}, key, value interface{}) bool
}

// Visit recursively visits all transformable object fields within the manifests and lets the visitor apply transformations/aggregations on them.
func (l *ManifestList) Visit(visitor FieldVisitor) (ManifestList, error) {
	var updated ManifestList

	for _, manifest := range *l {
		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(manifest, &m); err != nil {
			return nil, errors.Wrap(err, "reading Kubernetes YAML")
		}

		if len(m) == 0 {
			continue
		}

		traverseManifestFields(m, visitor)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling yaml")
		}

		updated = append(updated, updatedManifest)
	}

	return updated, nil
}

// traverseManifest traverses all transformable fields contained within the manifest.
func traverseManifestFields(manifest map[interface{}]interface{}, visitor FieldVisitor) {
	kind := manifest["kind"]
	if kind != "CustomResourceDefinition" {
		visitor = &recursiveVisitorDecorator{visitor}
	}
	visitFields(manifest, visitor)
}

// recursiveVisitorDecorator adds recursion to a FieldVisitor.
type recursiveVisitorDecorator struct {
	delegate FieldVisitor
}

func (d *recursiveVisitorDecorator) Visit(o map[interface{}]interface{}, k, v interface{}) bool {
	if d.delegate.Visit(o, k, v) {
		visitFields(v, d)
	}
	return false
}

// visitFields traverses all fields and calls the visitor for each.
func visitFields(o interface{}, visitor FieldVisitor) {
	switch entries := o.(type) {
	case []interface{}:
		for _, v := range entries {
			visitFields(v, visitor)
		}
	case map[interface{}]interface{}:
		for k, v := range entries {
			visitor.Visit(entries, k, v)
		}
	}
}
