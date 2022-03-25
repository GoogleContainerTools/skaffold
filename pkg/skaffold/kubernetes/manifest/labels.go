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

package manifest

import (
	"context"
	"strings"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

type ResourceSelectorLabels struct {
	allowlist map[apimachinery.GroupKind]latestV1.ResourceFilter
	denylist  map[apimachinery.GroupKind]latestV1.ResourceFilter
}

func NewResourceSelectorLabels(allowlist map[apimachinery.GroupKind]latestV1.ResourceFilter, denylist map[apimachinery.GroupKind]latestV1.ResourceFilter) *ResourceSelectorLabels {
	return &ResourceSelectorLabels{
		allowlist: allowlist,
		denylist:  denylist,
	}
}

func (rsi *ResourceSelectorLabels) allowByGroupKind(gk apimachinery.GroupKind) bool {
	if _, allowed := rsi.allowlist[gk]; allowed {
		// TODO(aaron-prindle) see if it makes sense to make this only use the allowlist...
		if rf, disallowed := rsi.denylist[gk]; disallowed {
			for _, s := range rf.Labels {
				if s == ".*" {
					return false
				}
			}
			for _, s := range rf.Image {
				if s == ".*" {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (rsi *ResourceSelectorLabels) allowByNavpath(gk apimachinery.GroupKind, navpath string, k string) (string, bool) {
	for _, w := range ConfigConnectorResourceSelector {
		if w.Matches(gk.Group, gk.Kind) {
			if k != metadataField {
				return "", false
			}
			return "labels", true
		}
	}

	if rf, ok := rsi.denylist[gk]; ok {
		for _, denypath := range rf.Labels {
			if denypath == ".*" {
				return "", false
			}
			// truncate the last part of the labels path and see if this matches
			lastDot := strings.LastIndex(denypath, ".")
			if lastDot == -1 {
				// no dot exists in denypath, denypath is invalid
				break
			}
			if navpath == denypath[:lastDot] {
				return "", false
			}
		}
	}

	if rf, ok := rsi.allowlist[gk]; ok {
		for _, allowpath := range rf.Labels {
			if allowpath == ".*" {
				if k != metadataField {
					return "", false
				}
				return "labels", true
			}
			// truncate the last part of the labels path and see if this matches
			lastDot := strings.LastIndex(allowpath, ".")
			if lastDot == -1 {
				// no dot exists in allowpath, allowpath is invalid
				continue
			}
			if navpath == allowpath[:lastDot] {
				return allowpath[lastDot+1:], true
			}
		}
	}
	return "", false
}

// SetLabels add labels to a list of Kubernetes manifests.
func (l *ManifestList) SetLabels(labels map[string]string, rs ResourceSelector) (ManifestList, error) {
	if len(labels) == 0 {
		return *l, nil
	}

	replacer := newLabelsSetter(labels)
	updated, err := l.Visit(replacer, rs)
	if err != nil {
		return nil, labelSettingErr(err)
	}

	log.Entry(context.TODO()).Debugln("manifests with labels", updated.String())

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

func (r *labelsSetter) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	labelsField, ok := rs.allowByNavpath(gk, navpath, k)
	if !ok {
		return true
	}

	if len(r.labels) == 0 {
		return false
	}
	metadata, ok := v.(map[string]interface{})
	if !ok {
		return true
	}

	l, present := metadata[labelsField]
	if !present {
		metadata[labelsField] = r.labels
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
