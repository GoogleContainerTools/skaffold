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
	jsonpatch "github.com/evanphx/json-patch/v5"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

type json6902Patch struct {
	raw       string          // raw original contents
	patch     jsonpatch.Patch // processed JSON 6902 patch
	matchInfo matchInfo       // used to match resources
}

func convertJSON6902Patches(patchesJSON6902 []config.PatchJSON6902) ([]json6902Patch, error) {
	patches := []json6902Patch{}
	for _, configPatch := range patchesJSON6902 {
		patchJSON, err := yaml.YAMLToJSON([]byte(configPatch.Patch))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		patch, err := jsonpatch.DecodePatch(patchJSON)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		patches = append(patches, json6902Patch{
			raw:       configPatch.Patch,
			patch:     patch,
			matchInfo: matchInfoForConfigJSON6902Patch(configPatch),
		})
	}
	return patches, nil
}
