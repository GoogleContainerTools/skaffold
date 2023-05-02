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
	"strings"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// KubeYAML takes a Kubernetes object YAML document stream to patch,
// merge patches, and JSON 6902 patches.
//
// It returns a patched a YAML document stream.
//
// Matching is performed on Kubernetes style v1 TypeMeta fields
// (kind and apiVersion), between the YAML documents and the patches.
//
// Patches match if their kind and apiVersion match a document, with the exception
// that if the patch does not set apiVersion it will be ignored.
func KubeYAML(toPatch string, patches []string, patches6902 []config.PatchJSON6902) (string, error) {
	// pre-process, including splitting up documents etc.
	resources, err := parseResources(toPatch)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse yaml to patch")
	}
	mergePatches, err := parseMergePatches(patches)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse patches")
	}
	json6902patches, err := convertJSON6902Patches(patches6902)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse JSON 6902 patches")
	}
	// apply patches and build result
	builder := &strings.Builder{}
	for i, r := range resources {
		// apply merge patches
		for _, p := range mergePatches {
			if _, err := r.applyMergePatch(p); err != nil {
				return "", errors.Wrap(err, "failed to apply patch")
			}
		}
		// apply RFC 6902 JSON patches
		for _, p := range json6902patches {
			if _, err := r.apply6902Patch(p); err != nil {
				return "", errors.Wrap(err, "failed to apply JSON 6902 patch")
			}
		}
		// write out result
		if err := r.encodeTo(builder); err != nil {
			return "", errors.Wrap(err, "failed to write patched resource")
		}
		// write document separator
		if i+1 < len(resources) {
			if _, err := builder.WriteString("---\n"); err != nil {
				return "", errors.Wrap(err, "failed to write document separator")
			}
		}
	}
	// verify that all patches were used
	return builder.String(), nil
}
