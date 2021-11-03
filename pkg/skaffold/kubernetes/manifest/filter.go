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

package manifest

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "sigs.k8s.io/yaml"
)

// Filter returns the manifest list filtered by the given selectors
func (l *ManifestList) Filter(selectors []GroupKindSelector) (ManifestList, error) {
	if l == nil {
		return nil, nil
	}
	var filtered ManifestList
	for _, yByte := range *l {
		// Convert yaml byte config to unstructured.Unstructured
		jByte, err := k8syaml.YAMLToJSON(yByte)
		if err != nil {
			return nil, fmt.Errorf("yaml to json error: %w", err)
		}
		var obj unstructured.Unstructured
		if err := obj.UnmarshalJSON(jByte); err != nil {
			return nil, fmt.Errorf("unmarshaling config: %w", err)
		}
		gvk := obj.GroupVersionKind()
		for _, w := range selectors {
			if w.Matches(gvk.Group, gvk.Kind) {
				filtered.Append(yByte)
			}
		}
	}
	return filtered, nil
}

// SelectResources returns the resources defined in the manifest list that match the given `GroupKindSelector` items
func (l *ManifestList) SelectResources(selectors ...GroupKindSelector) ([]unstructured.Unstructured, error) {
	if l == nil {
		return nil, nil
	}
	var customResources []unstructured.Unstructured
	for _, yByte := range *l {
		// Convert yaml byte config to unstructured.Unstructured
		jByte, err := k8syaml.YAMLToJSON(yByte)
		if err != nil {
			return nil, fmt.Errorf("yaml to json error: %w", err)
		}
		var obj unstructured.Unstructured
		if err := obj.UnmarshalJSON(jByte); err != nil {
			return nil, fmt.Errorf("unmarshaling config: %w", err)
		}

		gvk := obj.GroupVersionKind()
		for _, w := range selectors {
			if w.Matches(gvk.Group, gvk.Kind) {
				customResources = append(customResources, obj)
			}
		}
	}
	return customResources, nil
}
