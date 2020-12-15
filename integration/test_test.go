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

package integration

import (
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildAndTest(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	outputBytes := skaffold.Build("--quiet").InDir("examples/structure-tests").InNs(ns.Name).RunOrFailOutput(t)
	// Parse the Build Output
	buildArtifacts, err := flags.ParseBuildOutput(outputBytes)
	failNowIfError(t, err)
	if len(buildArtifacts.Builds) != 1 {
		t.Fatalf("expected 1 artifacts to be built, but found %d", len(buildArtifacts.Builds))
	}

	var skaffoldExampleTag string
	for _, a := range buildArtifacts.Builds {
		if a.ImageName == "skaffold-example" {
			skaffoldExampleTag = a.Tag
		}
	}
	if skaffoldExampleTag == "" {
		t.Fatalf("expected to find a tag for skaffold-example but found %s", skaffoldExampleTag)
	}

	tmpDir := testutil.NewTempDir(t)
	buildOutputFile := tmpDir.Path("build.out")
	tmpDir.Write("build.out", string(outputBytes))

	// Run Test using the build output
	skaffold.Test("--build-artifacts", buildOutputFile).InDir("examples/structure-tests").InNs(ns.Name).RunOrFail(t)
}

func TestTestWithInCorrectConfig(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	// We're not providing a tag for the getting-started image
	output, err := skaffold.Test().InDir("examples/getting-started").InNs(ns.Name).RunWithCombinedOutput(t)

	if err == nil {
		t.Errorf("expected to see an error since not every image tag is provided: %s", output)
	} else if !strings.Contains(string(output), "no tag provided for image [skaffold-example]") {
		t.Errorf("failed without saying the reason: %s", output)
	}
}
