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
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsNotTerminal(t *testing.T) {
	var w bytes.Buffer

	termFd, isTerm := IsTerminal(&w)

	testutil.CheckDeepEqual(t, uintptr(0x00), termFd)
	testutil.CheckDeepEqual(t, false, isTerm)
}
