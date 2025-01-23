/*
Copyright 2024 The Skaffold Authors

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
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

func init() {
	initCmd := exec.Command("tofu", "init")
	initCmd.Dir = "testdata/deploy-opentofu"
	initCmd.Run()
}

func TestTofuDeploy(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	output := string(skaffold.Deploy().InDir("testdata/deploy-opentofu").RunOrFailOutput(t))

	if !strings.Contains(output, "Apply complete!") {
		t.Fatal("Unexpectedly apply did not complete. Output was:", output)
	}

	if !strings.Contains(output, "google_addrs =") {
		t.Fatal("Output variable missing. Output was:", output)
	}
}
