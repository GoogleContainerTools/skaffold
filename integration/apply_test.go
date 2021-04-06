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

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDiagnoseRenderApply(t *testing.T) {
	// This test verifies that `skaffold apply` can consume the output of both `skaffold render` and `skaffold diagnose`.

	// 1. Run `skaffold diagnose --yaml-only` to resolve and combine skaffold configs for a multi-config project
	// 2. Run `skaffold render` using this config to hydrate manifests
	// 3. Run `skaffold apply` using the config from `diagnose` and the manifests from `render` to create resources on the cluster.

	testutil.Run(t, "DiagnoseRenderApply", func(t *testutil.T) {
		MarkIntegrationTest(t.T, NeedsGcp)
		ns, client := SetupNamespace(t.T)

		out := skaffold.Diagnose("--yaml-only").InDir("examples/multi-config-microservices").RunOrFailOutput(t.T)

		tmpDir := testutil.NewTempDir(t.T)
		tmpDir.Chdir()

		tmpDir.Write("skaffold-diagnose.yaml", string(out))

		out = skaffold.Render("--add-skaffold-labels=false", "-f", "skaffold-diagnose.yaml").InNs(ns.Name).RunOrFailOutput(t.T)
		tmpDir.Write("render.yaml", string(out))

		skaffold.Apply("render.yaml", "-f", "skaffold-diagnose.yaml").InNs(ns.Name).RunOrFail(t.T)

		depApp := client.GetDeployment("leeroy-app")
		t.CheckNotNil(depApp)

		depWeb := client.GetDeployment("leeroy-web")
		t.CheckNotNil(depWeb)
	})
}
