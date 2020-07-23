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

package deploy

import (
	"strings"
)

const (
	K8sManagedByLabelKey = "app.kubernetes.io/managed-by"
)

// DefaultLabeller adds K8s style managed-by label and a run-specific UUID annotation
type DefaultLabeller struct {
	addSkaffoldLabels bool
	customLabels      []string
}

func NewLabeller(addSkaffoldLabels bool, customLabels []string) *DefaultLabeller {
	return &DefaultLabeller{
		addSkaffoldLabels: addSkaffoldLabels,
		customLabels:      customLabels,
	}
}

func (d *DefaultLabeller) Labels() map[string]string {
	labels := map[string]string{}

	if d.addSkaffoldLabels {
		labels[K8sManagedByLabelKey] = "skaffold"
	}

	for _, cl := range d.customLabels {
		l := strings.SplitN(cl, "=", 2)
		if len(l) == 1 {
			labels[l[0]] = ""
			continue
		}
		labels[l[0]] = l[1]
	}

	return labels
}
