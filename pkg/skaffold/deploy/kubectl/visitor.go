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

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

// transformableAllowlist is the set of kinds that can be transformed by Skaffold.
var transformableAllowlist = map[apimachinery.GroupKind]bool{
	{Group: "", Kind: "Pod"}:                        true,
	{Group: "apps", Kind: "DaemonSet"}:              true,
	{Group: "apps", Kind: "Deployment"}:             true, // v1beta1, v1beta2: deprecated in K8s 1.9, removed in 1.16
	{Group: "apps", Kind: "ReplicaSet"}:             true,
	{Group: "apps", Kind: "StatefulSet"}:            true,
	{Group: "batch", Kind: "CronJob"}:               true,
	{Group: "batch", Kind: "Job"}:                   true,
	{Group: "extensions", Kind: "DaemonSet"}:        true, // v1beta1: deprecated in K8s 1.9, removed in 1.16
	{Group: "extensions", Kind: "Deployment"}:       true, // v1beta1: deprecated in K8s 1.9, removed in 1.16
	{Group: "extensions", Kind: "ReplicaSet"}:       true, // v1beta1: deprecated in K8s 1.9, removed in 1.16
	{Group: "serving.knative.dev", Kind: "Service"}: true,
	{Group: "agones.dev", Kind: "Fleet"}:            true,
	{Group: "agones.dev", Kind: "GameServer"}:       true,
}

// FieldVisitor represents the aggregation/transformation that should be performed on each traversed field.
type FieldVisitor interface {
	// Visit is called for each transformable key contained in the object and may apply transformations/aggregations on it.
	// It should return true to allow recursive traversal or false when the entry was transformed.
	Visit(object map[string]interface{}, key string, value interface{}) bool
}

// Visit recursively visits all transformable object fields within the manifests and lets the visitor apply transformations/aggregations on them.
func (l *ManifestList) Visit(visitor FieldVisitor) (ManifestList, error) {
	var updated ManifestList

	for _, manifest := range *l {
		m := make(map[string]interface{})
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
func traverseManifestFields(manifest map[string]interface{}, visitor FieldVisitor) {
	if shouldTransformManifest(manifest) {
		visitor = &recursiveVisitorDecorator{visitor}
	}
	visitFields(manifest, visitor)
}

func shouldTransformManifest(manifest map[string]interface{}) bool {
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

	return transformableAllowlist[groupKind]
}

// recursiveVisitorDecorator adds recursion to a FieldVisitor.
type recursiveVisitorDecorator struct {
	delegate FieldVisitor
}

func (d *recursiveVisitorDecorator) Visit(o map[string]interface{}, k string, v interface{}) bool {
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
	case map[string]interface{}:
		for k, v := range entries {
			visitor.Visit(entries, k, v)
		}
	}
}
