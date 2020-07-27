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
	"fmt"
	"sort"

	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// GetAllPodNamespaces lists the namespaces that should be watched.
// + The namespace passed on the command line
// + Current kube context's namespace
// + Namespaces referenced in Helm releases
func GetAllPodNamespaces(configNamespace string, cfg latest.Pipeline) ([]string, error) {
	nsMap := make(map[string]bool)

	if configNamespace == "" {
		// Get current kube context's namespace
		config, err := kubectx.CurrentConfig()
		if err != nil {
			return nil, fmt.Errorf("getting k8s configuration: %w", err)
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

	// Set additional namespaces each helm release referenced
	for _, namespace := range collectHelmReleasesNamespaces(cfg) {
		nsMap[namespace] = true
	}

	// Collate the slice of namespaces.
	namespaces := make([]string, 0, len(nsMap))
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}

	sort.Strings(namespaces)
	return namespaces, nil
}

func collectHelmReleasesNamespaces(cfg latest.Pipeline) []string {
	var namespaces []string

	if cfg.Deploy.HelmDeploy != nil {
		for _, release := range cfg.Deploy.HelmDeploy.Releases {
			if release.Namespace != "" {
				namespaces = append(namespaces, release.Namespace)
			}
		}
	}

	return namespaces
}
