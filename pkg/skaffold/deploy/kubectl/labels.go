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
func (l *ManifestList) SetLabelsAndAnnotations(labels, annotations map[string]string) (ManifestList, error) {
	replacer := newLabelsAndAnnotationsSetter(labels, annotations)

	updated, err := l.Visit(replacer)
	if err != nil {
		return nil, fmt.Errorf("setting labels: %w", err)
	}

	logrus.Debugln("manifests with labels", updated.String())

	return updated, nil
}

type labelsAndAnnotationsSetter struct {
	labels      map[string]string
	annotations map[string]string
}

func newLabelsAndAnnotationsSetter(labels, annotations map[string]string) *labelsAndAnnotationsSetter {
	return &labelsAndAnnotationsSetter{
		labels:      labels,
		annotations: annotations,
	}
}

func (r *labelsAndAnnotationsSetter) Visit(o map[string]interface{}, k string, v interface{}) bool {
	if k != "metadata" {
		return true
	}

	metadata, ok := v.(map[string]interface{})
	if !ok {
		return true
	}

	if visitAndReplace("labels", r.labels, &metadata) {
		return true
	}
	return visitAndReplace("annotations", r.annotations, &metadata)
}

func visitAndReplace(fieldName string, values map[string]string, metadata *map[string]interface{}) bool {
	if len(values) == 0 {
		return false
	}

	field, present := (*metadata)[fieldName]
	if !present {
		(*metadata)[fieldName] = values
		return false
	}

	existingValues, ok := field.(map[string]interface{})
	if !ok {
		return true
	}

	for k, v := range values {
		// Don't overwrite existing values
		if _, present := existingValues[k]; !present {
			existingValues[k] = v
		}
	}

	return false
}
