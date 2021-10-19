/*
Copyright 2021 The Skaffold Authors

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

package stringslice

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRemove(t *testing.T) {
	testutil.CheckDeepEqual(t, []string{""}, Remove([]string{""}, "ANY"))
	testutil.CheckDeepEqual(t, []string{"A", "B", "C"}, Remove([]string{"A", "B", "C"}, "ANY"))
	testutil.CheckDeepEqual(t, []string{"A", "C"}, Remove([]string{"A", "B", "C"}, "B"))
	testutil.CheckDeepEqual(t, []string{"B", "C"}, Remove([]string{"A", "B", "C"}, "A"))
	testutil.CheckDeepEqual(t, []string{"A", "C"}, Remove([]string{"A", "B", "B", "C"}, "B"))
	testutil.CheckDeepEqual(t, []string{}, Remove([]string{"B", "B"}, "B"))
}

func TestInsert(t *testing.T) {
	testutil.CheckDeepEqual(t, []string{"d", "e"}, Insert(nil, 0, []string{"d", "e"}))
	testutil.CheckDeepEqual(t, []string{"d", "e"}, Insert([]string{}, 0, []string{"d", "e"}))
	testutil.CheckDeepEqual(t, []string{"a", "d", "e", "b", "c"}, Insert([]string{"a", "b", "c"}, 1, []string{"d", "e"}))
	testutil.CheckDeepEqual(t, []string{"d", "e", "a", "b", "c"}, Insert([]string{"a", "b", "c"}, 0, []string{"d", "e"}))
	testutil.CheckDeepEqual(t, []string{"a", "b", "c", "d", "e"}, Insert([]string{"a", "b", "c"}, 3, []string{"d", "e"}))
	testutil.CheckDeepEqual(t, []string{"a", "b", "c"}, Insert([]string{"a", "b", "c"}, 0, nil))
	testutil.CheckDeepEqual(t, []string{"a", "b", "c"}, Insert([]string{"a", "b", "c"}, 1, nil))
}
