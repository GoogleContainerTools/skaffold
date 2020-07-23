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
)

// SetLabels add labels to a list of Kubernetes manifests.
func (l *ManifestList) SetLabels(labels map[string]string) (ManifestList, error) {
	if len(labels) == 0 {
		return *l, nil
	}

	replacer := newLabelsSetter(labels)
	updated, err := l.Visit(replacer)
	if err != nil {
		return nil, fmt.Errorf("setting labels in manifests: %w", err)
	}

	logrus.Debugln("manifests with labels", updated.String())

	return updated, nil
}

type labelsSetter struct {
	labels map[string]string
}

func newLabelsSetter(labels map[string]string) *labelsSetter {
	return &labelsSetter{
		labels: labels,
	}
}

func (r *labelsSetter) Visit(o map[string]interface{}, k string, v interface{}) bool {
	if k != "metadata" {
		return true
	}

	if len(r.labels) == 0 {
		return false
	}

	metadata, ok := v.(map[string]interface{})
	if !ok {
		return true
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
