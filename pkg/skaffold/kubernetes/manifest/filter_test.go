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
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFilter(t *testing.T) {
	testutil.Run(t, "TestFilter", func(t *testutil.T) {
		var manifests ManifestList
		manifests.Append([]byte(pod1 + "\n---\n" + pod2 + "\n---\n" + service))
		manifests, err := manifests.Filter(serviceSelector{})
		t.RequireNoError(err)
		t.CheckDeepEqual(service, string(manifests[0]))
	})
}

func TestSelectResources(t *testing.T) {
	testutil.Run(t, "TestSelectResources", func(t *testutil.T) {
		var manifests ManifestList
		manifests.Append([]byte(pod1 + "\n---\n" + pod2 + "\n---\n" + service))
		res, err := manifests.SelectResources(serviceSelector{})
		t.RequireNoError(err)
		t.CheckDeepEqual(1, len(res))
		t.CheckDeepEqual(schema.GroupVersionKind{Version: "v1", Kind: "Service"}, res[0].GetObjectKind().GroupVersionKind())
	})
}

type serviceSelector struct{}

func (serviceSelector) Matches(group, kind string) bool {
	return kind == "Service"
}
