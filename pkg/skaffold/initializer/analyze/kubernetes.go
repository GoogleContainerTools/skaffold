/*
Copyright 2020 The Skaffold Authors

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

package analyze

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
)

// kubeAnalyzer is a Visitor during the directory analysis that collects kubernetes manifests
type kubeAnalyzer struct {
	directoryAnalyzer
	kubernetesManifests []string
}

func (k *kubeAnalyzer) analyzeFile(filePath string) error {
	if kubernetes.IsKubernetesManifest(filePath) && !schema.IsSkaffoldConfig(filePath) {
		k.kubernetesManifests = append(k.kubernetesManifests, filePath)
	}
	return nil
}
