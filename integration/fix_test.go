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

package integration

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

func TestFix(t *testing.T) {
	// TODO: This test shall pass once render v2 is completed.
	t.SkipNow()

	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	out := skaffold.Fix().InDir("testdata/fix").RunOrFailOutput(t)

	skaffold.Run().WithConfig("-").InDir("testdata/fix").InNs(ns.Name).WithStdin(out).RunOrFail(t)
}
