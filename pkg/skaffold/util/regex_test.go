/*
Copyright 2020 The Skaffold Authors

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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKubectxEqual(t *testing.T) {
	ctxRe := ".*-i.*-am.*-test.*"
	ctxRePos := "wohoo-i-am-test-or"
	ctxReNeg := "test-am-i"
	testutil.CheckDeepEqual(t, true, RegexEqual(ctxRe, ctxRePos))
	testutil.CheckDeepEqual(t, false, RegexEqual(ctxRe, ctxReNeg))

	ctxStr := "^s^"
	ctxStrPos := "^s^"
	ctxStrNeg := "test-am-i"
	testutil.CheckDeepEqual(t, true, RegexEqual(ctxStr, ctxStrPos))
	testutil.CheckDeepEqual(t, false, RegexEqual(ctxStr, ctxStrNeg))
}
