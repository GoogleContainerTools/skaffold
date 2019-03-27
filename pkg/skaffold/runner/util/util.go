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

package util

import (
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/pkg/errors"
)

func GetAllPodNamespaces(configNamespace string) ([]string, error) {
	// We also get the default namespace.
	nsMap := make(map[string]bool)
	if configNamespace == "" {
		config, err := kubectx.CurrentConfig()
		if err != nil {
			return nil, errors.Wrap(err, "getting k8s configuration")
		}
		context, ok := config.Contexts[config.CurrentContext]
		if ok {
			nsMap[context.Namespace] = true
		} else {
			nsMap[""] = true
		}
	} else {
		nsMap[configNamespace] = true
	}

	// FIXME: Set additional namespaces from the selected yamls.

	// Collate the slice of namespaces.
	namespaces := make([]string, 0, len(nsMap))
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}
