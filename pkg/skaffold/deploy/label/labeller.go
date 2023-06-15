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

package label

import (
	"fmt"
	"strings"
)

const (
	RunIDLabel          = "skaffold.dev/run-id"
	DebugContainerLabel = "skaffold.dev/debug"
)

type Config interface {
	RunIDSelector() string
}

// DefaultLabeller adds a run-specific UUID label
type DefaultLabeller struct {
	addSkaffoldLabels bool
	customLabels      []string
	runID             string
}

func NewLabeller(addSkaffoldLabels bool, customLabels []string, runID string) *DefaultLabeller {
	return &DefaultLabeller{
		addSkaffoldLabels: addSkaffoldLabels,
		customLabels:      customLabels,
		runID:             runID,
	}
}

func (d *DefaultLabeller) Labels() map[string]string {
	labels := map[string]string{}

	if d.addSkaffoldLabels {
		labels[RunIDLabel] = d.runID
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

func (d *DefaultLabeller) DebugLabels() map[string]string {
	labels := d.Labels()
	labels[DebugContainerLabel] = "true"

	return labels
}

func (d *DefaultLabeller) RunIDSelector() string {
	return fmt.Sprintf("%s=%s", RunIDLabel, d.Labels()[RunIDLabel])
}

func (d *DefaultLabeller) GetRunID() string {
	return d.runID
}
