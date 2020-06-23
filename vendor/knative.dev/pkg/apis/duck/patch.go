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

	jsonmergepatch "github.com/evanphx/json-patch"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
)

func marshallBeforeAfter(before, after interface{}) ([]byte, []byte, error) {
	rawBefore, err := json.Marshal(before)
	if err != nil {
		return nil, nil, err
	}

	rawAfter, err := json.Marshal(after)
	if err != nil {
		return rawBefore, nil, err
	}

	return rawBefore, rawAfter, nil
}

// CreateMergePatch creates a json merge patch as specified in
// http://tools.ietf.org/html/draft-ietf-appsawg-json-merge-patch-07
func CreateMergePatch(before, after interface{}) ([]byte, error) {
	rawBefore, rawAfter, err := marshallBeforeAfter(before, after)
	if err != nil {
		return nil, err
	}
	return jsonmergepatch.CreateMergePatch(rawBefore, rawAfter)
}

// CreateBytePatch is a helper function that creates the same content as
// CreatePatch, but returns in []byte format instead of JSONPatch.
func CreateBytePatch(before, after interface{}) ([]byte, error) {
	patch, err := CreatePatch(before, after)
	if err != nil {
		return nil, err
	}
	return patch.MarshalJSON()
}

// CreatePatch creates a patch as specified in http://jsonpatch.com/
func CreatePatch(before, after interface{}) (JSONPatch, error) {
	rawBefore, rawAfter, err := marshallBeforeAfter(before, after)
	if err != nil {
		return nil, err
	}
	return jsonpatch.CreatePatch(rawBefore, rawAfter)
}

type JSONPatch []jsonpatch.JsonPatchOperation

func (p JSONPatch) MarshalJSON() ([]byte, error) {
	return json.Marshal([]jsonpatch.JsonPatchOperation(p))
}
