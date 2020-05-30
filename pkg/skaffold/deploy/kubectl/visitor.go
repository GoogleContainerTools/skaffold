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
	"fmt"

	yaml "gopkg.in/yaml.v2"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
)

// transformableWhitelist is the set of kinds that can be transformed by Skaffold.
var transformableWhitelist = map[apimachinery.GroupKind]bool{
	{Group: "", Kind: "Pod"}:                        true,
	{Group: "apps", Kind: "DaemonSet"}:              true,
	{Group: "apps", Kind: "Deployment"}:             true,
	{Group: "apps", Kind: "ReplicaSet"}:             true,
	{Group: "apps", Kind: "StatefulSet"}:            true,
	{Group: "batch", Kind: "CronJob"}:               true,
	{Group: "batch", Kind: "Job"}:                   true,
	{Group: "serving.knative.dev", Kind: "Service"}: true,
}

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
			return nil, fmt.Errorf("reading Kubernetes YAML: %w", err)
		}

		if len(m) == 0 {
			continue
		}

		traverseManifestFields(m, visitor)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("marshalling yaml: %w", err)
		}

		updated = append(updated, updatedManifest)
	}

	return updated, nil
}

// traverseManifest traverses all transformable fields contained within the manifest.
func traverseManifestFields(manifest map[interface{}]interface{}, visitor FieldVisitor) {
	if shouldTransformManifest(manifest) {
		visitor = &recursiveVisitorDecorator{visitor}
	}
	visitFields(manifest, visitor)
}

func shouldTransformManifest(manifest map[interface{}]interface{}) bool {
	var apiVersion string
	switch value := manifest["apiVersion"].(type) {
	case string:
		apiVersion = value
	default:
		return false
	}

	var kind string
	switch value := manifest["kind"].(type) {
	case string:
		kind = value
	default:
		return false
	}

	gvk := apimachinery.FromAPIVersionAndKind(apiVersion, kind)
	groupKind := apimachinery.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}

	return transformableWhitelist[groupKind]
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
