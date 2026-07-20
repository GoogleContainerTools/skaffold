/*
Copyright The Kubernetes Authors.

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
	"encoding/json"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
)

// mergeAndStrip performs a custom merge of kubeadm fields (like extraArgs and certSANs)
// between targetJSON and patchJSON, taking targetVersion into account.
// It strips those fields from the patch so that the standard merge patch doesn't overwrite them.
func mergeAndStrip(targetJSON, patchJSON []byte, targetVersion, patchVersion string) ([]byte, []byte, error) {
	var target map[string]interface{}
	if err := json.Unmarshal(targetJSON, &target); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	var patch map[string]interface{}
	if err := json.Unmarshal(patchJSON, &patch); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	if err := walkAndMerge(target, patch, targetVersion, patchVersion); err != nil {
		return nil, nil, err
	}

	newTargetJSON, err := json.Marshal(target)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	strippedPatchJSON, err := json.Marshal(patch)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return newTargetJSON, strippedPatchJSON, nil
}

// walkAndMerge recursively walks the patch map and applies custom merging for
// extraArgs, kubeletExtraArgs, and certSANs, and deletes them from the patch map.
func walkAndMerge(target, patch map[string]interface{}, targetVersion, patchVersion string) error {
	for k, patchVal := range patch {
		if k == "extraArgs" || k == "kubeletExtraArgs" {
			if strings.HasSuffix(targetVersion, "v1beta4") {
				if err := mergeExtraArgs(target, patch, k, patchVersion); err != nil {
					return err
				}
				delete(patch, k)
			}
			continue
		}
		if k == "certSANs" {
			if err := mergeCertSANs(target, patch, k); err != nil {
				return err
			}
			delete(patch, k)
			continue
		}

		// Recurse if the value is a map
		if patchMap, ok := patchVal.(map[string]interface{}); ok {
			targetVal, exists := target[k]
			if !exists {
				targetMap := make(map[string]interface{})
				target[k] = targetMap
				targetVal = targetMap
			}
			if targetMap, ok := targetVal.(map[string]interface{}); ok {
				if err := walkAndMerge(targetMap, patchMap, targetVersion, patchVersion); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mergeExtraArgs(target, patch map[string]interface{}, key string, patchVersion string) error {
	patchVal := patch[key]
	var patchSlice []interface{}
	if slice, ok := patchVal.([]interface{}); ok {
		patchSlice = slice
	} else if patchVersion == "" {
		if patchMap, ok := patchVal.(map[string]interface{}); ok {
			patchSlice = convertOldExtraArgsToNew(patchMap)
		}
	}

	targetVal := target[key]
	var targetSlice []interface{}
	if targetVal != nil {
		if slice, ok := targetVal.([]interface{}); ok {
			targetSlice = slice
		} else if targetMap, ok := targetVal.(map[string]interface{}); ok {
			targetSlice = convertOldExtraArgsToNew(targetMap)
		}
	}

	for _, pItemVal := range patchSlice {
		pItem, ok := pItemVal.(map[string]interface{})
		if !ok {
			continue
		}
		nameVal, ok := pItem["name"]
		if !ok {
			continue
		}
		name, ok := nameVal.(string)
		if !ok {
			continue
		}
		value := pItem["value"]

		// Find in target
		foundIdx := -1
		for idx, tItemVal := range targetSlice {
			tItem, ok := tItemVal.(map[string]interface{})
			if !ok {
				continue
			}
			tNameVal, ok := tItem["name"]
			if !ok {
				continue
			}
			tName, ok := tNameVal.(string)
			if !ok {
				continue
			}
			if tName == name {
				foundIdx = idx
				break
			}
		}

		if foundIdx >= 0 {
			if value == nil {
				// delete
				targetSlice = append(targetSlice[:foundIdx], targetSlice[foundIdx+1:]...)
			} else {
				// update
				if tItem, ok := targetSlice[foundIdx].(map[string]interface{}); ok {
					tItem["value"] = value
				}
			}
		} else {
			if value != nil {
				// append
				targetSlice = append(targetSlice, map[string]interface{}{
					"name":  name,
					"value": value,
				})
			}
		}
	}
	target[key] = targetSlice
	return nil
}

func convertOldExtraArgsToNew(old map[string]interface{}) []interface{} {
	var res []interface{}
	for k, v := range old {
		res = append(res, map[string]interface{}{
			"name":  k,
			"value": v,
		})
	}
	return res
}

func mergeCertSANs(target, patch map[string]interface{}, key string) error {
	targetVal := target[key]
	var targetSlice []interface{}
	if targetVal != nil {
		if slice, ok := targetVal.([]interface{}); ok {
			targetSlice = slice
		}
	}

	patchVal := patch[key]
	var patchSlice []interface{}
	if patchVal != nil {
		if slice, ok := patchVal.([]interface{}); ok {
			patchSlice = slice
		}
	}

	for _, pItem := range patchSlice {
		exists := false
		for _, tItem := range targetSlice {
			if tItem == pItem {
				exists = true
				break
			}
		}
		if !exists {
			targetSlice = append(targetSlice, pItem)
		}
	}
	target[key] = targetSlice
	return nil
}
