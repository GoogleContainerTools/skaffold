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
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

const namespaceField = "namespace"

const defaultNamespace = "default"

// CollectNamespaces returns all the namespaces in the manifests.
func (l *ManifestList) CollectNamespaces() ([]string, error) {
	replacer := newNamespaceCollector()

	// TODO(aaron-prindle) make sure this is ok?
	rs := &ResourceSelectorImages{}
	if _, err := l.Visit(replacer, rs); err != nil {
		// if _, err := l.Visit(replacer, make(map[schema.GroupKind]latest.ResourceFilter), make(map[schema.GroupKind]latest.ResourceFilter)); err != nil {
		// TODO(aaron-prindle) verify this doesn't need to support allow/deny list, also see if 'nil' is better option for unused inputs
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

func (r *namespaceCollector) Visit(gk schema.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if k != metadataField {
		return true
	}

	metadata, ok := v.(map[string]interface{})
	if !ok {
		return true
	}
	if nsValue, present := metadata[namespaceField]; present {
		nsString, ok := nsValue.(string)
		if !ok || nsString == "" {
			return true
		}
		if ns := strings.TrimSpace(nsString); ns != "" {
			r.namespaces[ns] = true
		}
	}
	return false
}

// SetNamespace sets labels to a list of Kubernetes manifests if they are not set.
// Returns error if any manifest in the list has namespace set.
func (l *ManifestList) SetNamespace(namespace string, rs ResourceSelector) (ManifestList, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}
	var updated ManifestList
	for _, item := range *l {
		updatedManifest := item
		m := make(map[string]interface{})
		if err := yaml.Unmarshal(item, &m); err != nil {
			return nil, fmt.Errorf("reading Kubernetes YAML: %w", err)
		}
		if shouldTransformManifest(m, rs) {
			var errU error
			if errU = addOrUpdateNamespace(m, namespace); errU != nil {
				return nil, errU
			}
			updatedManifest, errU = yaml.Marshal(m)
			if errU != nil {
				return nil, nsSettingErr(errU)
			}
		}
		updated = append(updated, updatedManifest)
	}

	log.Entry(context.TODO()).Debugln("manifests set with namespace", updated.String())
	return updated, nil
}

func addOrUpdateNamespace(manifest map[string]interface{}, ns string) error {
	originalMetadata, ok := manifest[metadataField]
	if !ok {
		metadataAdded := make(map[string]interface{})
		metadataAdded[namespaceField] = ns
		manifest[metadataField] = metadataAdded
		return nil
	}
	metadata, ok := originalMetadata.(map[string]interface{})
	if !ok {
		return nsSettingErr(fmt.Errorf("error converting %s to map[string]interface{}", originalMetadata))
	}
	nsValue, present := metadata[namespaceField]
	if !present || isEmptyOrEqual(nsValue, ns) {
		metadata[namespaceField] = ns
		return nil
	}

	if present && isEmptyOrEqual(ns, defaultNamespace) {
		return nil
	}

	warnings.Printf("a manifest already has namespace set \"%s\" which conflicts with namespace on the CLI \"%s\"", nsValue, ns)
	return nil
}

func isEmptyOrEqual(v interface{}, s string) bool {
	// check if namespace is set to empty string
	if v == nil {
		return true
	}
	nsString, ok := v.(string)
	if !ok {
		return false
	}
	return nsString == "" || nsString == s
}
