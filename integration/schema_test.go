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

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSchema(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	testutil.Run(t, "list", func(t *testutil.T) {
		out := skaffold.Schema("list").RunOrFailOutput(t.T)

		t.CheckContains("skaffold/v1", string(out))
	})

	testutil.Run(t, "get", func(t *testutil.T) {
		out := skaffold.Schema("get", "skaffold/v1").RunOrFailOutput(t.T)

		t.CheckContains("#/definitions/SkaffoldConfig", string(out))
	})
}
