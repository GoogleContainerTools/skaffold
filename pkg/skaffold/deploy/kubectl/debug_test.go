/*
Copyright 2018 The Skaffold Authors

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

package kubectl

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTraverse(t *testing.T) {
	manifest := map[interface{}]interface{}{
		"string": "value1",
		"node1": map[interface{}]interface{}{
			"node2": "value2",
		},
	}
	result, ok := traverse(manifest, "string")
	testutil.CheckDeepEqual(t, true, ok)
	testutil.CheckDeepEqual(t, "value1", result)

	result, ok = traverse(manifest, "node1")
	testutil.CheckDeepEqual(t, true, ok)
	testutil.CheckDeepEqual(t, manifest["node1"], result)

	result, ok = traverse(manifest, "node1", "node2")
	testutil.CheckDeepEqual(t, true, ok)
	testutil.CheckDeepEqual(t, "value2", result)

	result, ok = traverse(manifest, "foo")
	testutil.CheckDeepEqual(t, false, ok)

	result, ok = traverse(manifest, "node1", "foo")
	testutil.CheckDeepEqual(t, false, ok)
}
