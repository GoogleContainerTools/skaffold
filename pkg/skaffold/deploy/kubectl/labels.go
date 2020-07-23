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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// SetLabels add labels to a list of Kubernetes manifests.
func (l *ManifestList) SetLabels(addRunIDAnnotation bool, runID string, labels map[string]string) (ManifestList, error) {
	if !addRunIDAnnotation {
		// empty run-id will skip setting the annotation altogether
		runID = ""
	}
	replacer := newLabelsSetter(runID, labels)
	updated, err := l.Visit(replacer)
	if err != nil {
		return nil, fmt.Errorf("setting labels in manifests: %w", err)
	}

	logrus.Debugln("manifests with labels", updated.String())

	return updated, nil
}

type labelsSetter struct {
	runID  string
	labels map[string]string
}

func newLabelsSetter(runID string, labels map[string]string) *labelsSetter {
	return &labelsSetter{
		runID:  runID,
		labels: labels,
	}
}

func (r *labelsSetter) Visit(o map[string]interface{}, k string, v interface{}) bool {
	if k != "metadata" {
		return true
	}

	metadata, ok := v.(map[string]interface{})
	if !ok {
		return true
	}

	if r.runID != "" {
		a, present := metadata["annotations"]
		if !present {
			metadata["annotations"] = map[string]string{constants.RunIDAnnotation: r.runID}
		} else {
			annotations, ok := a.(map[string]interface{})
			if !ok {
				return true
			}
			annotations[constants.RunIDAnnotation] = r.runID
		}
	}

	if len(r.labels) == 0 {
		return false
	}

	l, present := metadata["labels"]
	if !present {
		metadata["labels"] = r.labels
		return false
	}

	labels, ok := l.(map[string]interface{})
	if !ok {
		return true
	}

	for k, v := range r.labels {
		// Don't overwrite existing labels
		if _, present := labels[k]; !present {
			labels[k] = v
		}
	}

	return false
}
