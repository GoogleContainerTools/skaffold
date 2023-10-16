/*
Copyright 2019 The Kubernetes Authors.

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

package patch

import (
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// we match resources and patches on their v1 TypeMeta
type matchInfo struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

func parseYAMLMatchInfo(raw string) (matchInfo, error) {
	m := matchInfo{}
	if err := yaml.Unmarshal([]byte(raw), &m); err != nil {
		return matchInfo{}, errors.Wrapf(err, "failed to parse type meta for %q", raw)
	}
	return m, nil
}

func matchInfoForConfigJSON6902Patch(patch config.PatchJSON6902) matchInfo {
	return matchInfo{
		Kind:       patch.Kind,
		APIVersion: groupVersionToAPIVersion(patch.Group, patch.Version),
	}
}

func groupVersionToAPIVersion(group, version string) string {
	if group == "" {
		return version
	}
	return group + "/" + version
}
