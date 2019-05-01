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

package runner

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

const (
	K8ManagedByLabel = "app.kubernetes.io/managed-by"
)

// DefaultLabeller adds K9 style managed-by label
type DefaultLabeller struct {
	version string
}

func NewLabeller(verStr string) *DefaultLabeller {
	if verStr == "" {
		verStr = version.Get().Version
	}
	return &DefaultLabeller{
		version: verStr,
	}
}

func (d *DefaultLabeller) Labels() map[string]string {
	return map[string]string{
		K8ManagedByLabel: fmt.Sprintf("skaffold-%s", d.version),
	}
}
