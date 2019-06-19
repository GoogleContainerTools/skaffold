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

package testutil

import "testing"

var (
	strVariable = "original"
	fnVariable  = func() string { return "original" }
)

func TestOverride(t *testing.T) {
	restore := Override(t, &strVariable, "temporary")
	CheckDeepEqual(t, "temporary", strVariable)

	restore()
	CheckDeepEqual(t, "original", strVariable)
}

func TestOverrideFunction(t *testing.T) {
	restore := Override(t, &fnVariable, func() string { return "temporary" })
	CheckDeepEqual(t, "temporary", fnVariable())

	restore()
	CheckDeepEqual(t, "original", fnVariable())
}

func TestOverrideToNil(t *testing.T) {
	restore := Override(t, &fnVariable, nil)
	CheckDeepEqual(t, true, fnVariable == nil)

	restore()
	CheckDeepEqual(t, "original", fnVariable())
}
