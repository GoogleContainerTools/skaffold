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

import apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

// resourceSelectorImagePullPolicy selects PodSpecs for transforming the imagePullPolicy field
// based on allowlist and denylist rules for their GroupKind and navigation path.
type resourceSelectorImagePullPolicy struct{}

func NewResourceSelectorImagePullPolicy() ResourceSelector {
	return &resourceSelectorImagePullPolicy{}
}

// allowByGroupKind checks if a GroupKind is allowed for transformation.
// It allows GroupKinds in the allowlist unless they are in the denylist with ".*" in PodSpec paths, which blocks all PodSpecs.
func (rs *resourceSelectorImagePullPolicy) allowByGroupKind(gk apimachinery.GroupKind) bool {
	if _, allowed := TransformAllowlist[gk]; allowed {
		if rf, disallowed := TransformDenylist[gk]; disallowed {
			for _, s := range rf.PodSpec {
				if s == ".*" {
					return false
				}
			}
		}
		return true
	}
	return false
}

// allowByNavpath checks if a GroupKind's PodSpec path (navpath) allows transformation.
// It blocks transformation if the path matches a denylist entry or ".*".
// If not denied, it permits transformation if the path matches an allowlist entry or ".*".
func (rs *resourceSelectorImagePullPolicy) allowByNavpath(gk apimachinery.GroupKind, navpath string, k string) (string, bool) {
	if rf, ok := TransformDenylist[gk]; ok {
		for _, denypath := range rf.PodSpec {
			if denypath == ".*" || navpath == denypath {
				return "", false
			}
		}
	}
	if rf, ok := TransformAllowlist[gk]; ok {
		for _, allowpath := range rf.PodSpec {
			if allowpath == ".*" || navpath == allowpath {
				return "", true
			}
		}
	}
	return "", false
}

func (l *ManifestList) ReplaceImagePullPolicy(rs ResourceSelector) (ManifestList, error) {
	r := &imagePullPolicyReplacer{}
	return l.Visit(r, rs)
}

// imagePullPolicyReplacer implements FieldVisitor and modifies the "imagePullPolicy" field in Kubernetes manifests.
type imagePullPolicyReplacer struct{}

// Visit sets the value of the "imagePullPolicy" field in a Kubernetes manifest to "Never".
func (i *imagePullPolicyReplacer) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	const imagePullPolicyField = "imagePullPolicy"
	if _, allowed := rs.allowByNavpath(gk, navpath, k); !allowed {
		return true
	}
	if k != imagePullPolicyField {
		return true
	}
	if _, ok := v.(string); !ok {
		return true
	}
	o[imagePullPolicyField] = "Never"
	return false
}
