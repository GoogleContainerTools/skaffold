/*
Copyright 2018 The Knative Authors

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

package duck

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ToUnstructured takes an instance of a OneOfOurs compatible type and
// converts it to unstructured.Unstructured.  We take OneOfOurs in place
// or runtime.Object because sometimes we get resources that do not have their
// TypeMeta populated but that is required for unstructured.Unstructured to
// deserialize things, so we leverage our content-agnostic GroupVersionKind()
// method to populate this as-needed (in a copy, so that we don't modify the
// informer's copy, if that is what we are passed).
func ToUnstructured(desired OneOfOurs) (*unstructured.Unstructured, error) {
	// If the TypeMeta is not populated, then unmarshalling will fail, so ensure
	// the TypeMeta is populated.  See also EnsureTypeMeta.
	if gvk := desired.GroupVersionKind(); gvk.Version == "" || gvk.Kind == "" {
		gvk = desired.GetGroupVersionKind()
		desired = desired.DeepCopyObject().(OneOfOurs)
		desired.SetGroupVersionKind(gvk)
	}

	// Convert desired to unstructured.Unstructured
	b, err := json.Marshal(desired)
	if err != nil {
		return nil, err
	}
	ud := &unstructured.Unstructured{}
	if err := json.Unmarshal(b, ud); err != nil {
		return nil, err
	}
	return ud, nil
}

// FromUnstructured takes unstructured object from (say from client-go/dynamic) and
// converts it into our duck types.
func FromUnstructured(obj json.Marshaler, target interface{}) error {
	// Use the unstructured marshaller to ensure it's proper JSON
	raw, err := obj.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &target)
}
