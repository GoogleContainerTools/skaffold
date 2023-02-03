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

package integration

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

func TestEnvFile(t *testing.T) {
	// This test verifies that Skaffold can load environment variables from the `skaffold.env` file.
	// In the project `examples/using-env-file` run `skaffold run` and verify that environment variables were loaded from the `skaffold.env` file

	MarkIntegrationTest(t, CanRunWithoutGcp)
	ns, _ := SetupNamespace(t)
	out := skaffold.Run("--tail").InDir("examples/using-env-file").InNs(ns.Name).RunLive(t)

	WaitForLogs(t, out, "Hello from Skaffold!")
	skaffold.Delete().InDir("examples/using-env-file").InNs(ns.Name).RunOrFail(t)
}
