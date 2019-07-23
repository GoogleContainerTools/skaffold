/*
 Copyright 2019 Knative Authors LLC
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

package merge

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// CombineStepsWithStepTemplate takes a possibly nil container template and a list of step containers, merging each
// of the step containers onto the container template, if it's not nil, and returning the resulting list.
func CombineStepsWithStepTemplate(template *v1.Container, steps []v1.Container) ([]v1.Container, error) {
	if template == nil {
		return steps, nil
	}

	// We need JSON bytes to generate a patch to merge the step containers onto the template container, so marshal the template.
	templateAsJSON, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	// We need to do a three-way merge to actually combine the template and step containers, so we need an empty container
	// as the "original"
	emptyAsJSON, err := json.Marshal(&v1.Container{})
	if err != nil {
		return nil, err
	}

	for i, s := range steps {
		// Marshal the step to JSON
		stepAsJSON, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}

		// Get the patch meta for Container, which is needed for generating and applying the merge patch.
		patchSchema, err := strategicpatch.NewPatchMetaFromStruct(template)

		if err != nil {
			return nil, err
		}

		// Create a merge patch, with the empty JSON as the original, the step JSON as the modified, and the template
		// JSON as the current - this lets us do a deep merge of the template and step containers, with awareness of
		// the "patchMerge" tags.
		patch, err := strategicpatch.CreateThreeWayMergePatch(emptyAsJSON, stepAsJSON, templateAsJSON, patchSchema, true)
		if err != nil {
			return nil, err
		}

		// Actually apply the merge patch to the template JSON.
		mergedAsJSON, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(templateAsJSON, patch, patchSchema)
		if err != nil {
			return nil, err
		}

		// Unmarshal the merged JSON to a Container pointer, and return it.
		merged := &v1.Container{}
		err = json.Unmarshal(mergedAsJSON, merged)
		if err != nil {
			return nil, err
		}

		// If the container's args is nil, reset it to empty instead
		if merged.Args == nil && s.Args != nil {
			merged.Args = []string{}
		}

		steps[i] = *merged
	}
	return steps, nil
}
