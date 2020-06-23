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
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

const (
	K8sManagedByLabelKey = "app.kubernetes.io/managed-by"
	RunIDLabel           = "skaffold.dev/run-id"
	unknownVersion       = "unknown"
	empty                = ""
)

var runID = uuid.New().String()

// DefaultLabeller adds K8s style managed-by label and a run-specific UUID label
type DefaultLabeller struct {
	opts    config.SkaffoldOptions
	version string
	runID   string
}

func NewLabeller(opts config.SkaffoldOptions) *DefaultLabeller {
	verStr := version.Get().Version
	if verStr == empty {
		verStr = unknownVersion
	}
	return &DefaultLabeller{
		opts:    opts,
		version: verStr,
		runID:   runID,
	}
}

func (d *DefaultLabeller) Labels() map[string]string {
	labels := map[string]string{
		K8sManagedByLabelKey: fmt.Sprintf("skaffold-%s", d.version),
		RunIDLabel:           d.runID,
	}

	if d.opts.Cleanup {
		labels["skaffold.dev/cleanup"] = "true"
	}
	if d.opts.Tail {
		labels["skaffold.dev/tail"] = "true"
	}
	if d.opts.Namespace != "" {
		labels["skaffold.dev/namespace"] = d.opts.Namespace
	}
	for i, profile := range d.opts.Profiles {
		key := fmt.Sprintf("skaffold.dev/profile.%d", i)
		labels[key] = profile
	}
	for _, cl := range d.opts.CustomLabels {
		l := strings.SplitN(cl, "=", 2)
		if len(l) == 1 {
			labels[l[0]] = ""
			continue
		}
		labels[l[0]] = l[1]
	}
	return labels
}

func (d *DefaultLabeller) RunIDSelector() string {
	return fmt.Sprintf("%s=%s", RunIDLabel, d.Labels()[RunIDLabel])
}
