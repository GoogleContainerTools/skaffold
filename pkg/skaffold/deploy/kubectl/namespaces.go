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
	"sort"
	"strings"
)

// CollectNamespaces returns all the namespaces in the manifests.
func (l *ManifestList) CollectNamespaces() ([]string, error) {
	replacer := newNamespaceCollector()

	if _, err := l.Visit(replacer); err != nil {
		return nil, fmt.Errorf("collecting namespaces: %w", err)
	}

	namespaces := make([]string, 0, len(replacer.namespaces))
	for ns := range replacer.namespaces {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}

type namespaceCollector struct {
	namespaces map[string]bool
}

func newNamespaceCollector() *namespaceCollector {
	return &namespaceCollector{
		namespaces: map[string]bool{},
	}
}

func (r *namespaceCollector) Visit(o map[string]interface{}, k string, v interface{}) bool {
	if k != "metadata" {
		return true
	}

	metadata, ok := v.(map[string]interface{})
	if !ok {
		return true
	}
	if nsValue, present := metadata["namespace"]; present {
		if ns := strings.TrimSpace(nsValue.(string)); ns != "" {
			r.namespaces[ns] = true
		}
	}
	return false
}
