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

package initializer

import (
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
)

// IsSupportedKubernetesFileExtension is for determining if a file under a glob pattern
// is deployable file format. It makes no attempt to check whether or not the file
// is actually deployable or has the correct contents.
func IsSupportedKubernetesFileExtension(n string) bool {
	for _, s := range kubectl.ValidSuffixes {
		if strings.HasSuffix(n, s) {
			return true
		}
	}
	return false
}

// IsSkaffoldConfig is for determining if a file is skaffold config file.
func IsSkaffoldConfig(file string) bool {
	if config, err := schema.ParseConfig(file, false); err == nil && config != nil {
		return true
	}
	return false
}
