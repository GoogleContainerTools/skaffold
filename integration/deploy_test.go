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
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildDeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	outputBytes := skaffold.Build("--quiet").InDir("examples/microservices").InNs(ns.Name).RunOrFailOutput(t)
	// Parse the Build Output
	buildArtifacts, err := flags.ParseBuildOutput(outputBytes)
	if err != nil {
		t.Fatalf("Unparsable build output %s", string(outputBytes))
	}
	if len(buildArtifacts.Builds) != 2 {
		t.Fatalf("expected 2 artifacts to be built, but found %d", len(buildArtifacts.Builds))
	}

	var webTag, appTag string
	for _, a := range buildArtifacts.Builds {
		if a.ImageName == "gcr.io/k8s-skaffold/leeroy-web" {
			webTag = a.Tag
		}
		if a.ImageName == "gcr.io/k8s-skaffold/leeroy-app" {
			appTag = a.Tag
		}
	}
	if webTag == "" {
		t.Fatalf("expected to find a tag for leeroy-web but found none %s", webTag)
	}
	if appTag == "" {
		t.Fatalf("expected to find a tag for leeroy-app but found none %s", appTag)
	}

	dir, cleanUp := testutil.NewTempDir(t)
	buildOutputFile := dir.Path("build.out")
	defer cleanUp()
	dir.Write("build.out", string(outputBytes))

	// Run Deploy using the build output
	// See https://github.com/GoogleContainerTools/skaffold/issues/2372 on why status-check=false
	skaffold.Deploy("--build-artifacts", buildOutputFile, "--status-check=false").InDir("examples/microservices").InNs(ns.Name).RunOrFail(t)

	depApp := client.GetDeployment("leeroy-app")
	testutil.CheckDeepEqual(t, appTag, depApp.Spec.Template.Spec.Containers[0].Image)

	depWeb := client.GetDeployment("leeroy-web")
	testutil.CheckDeepEqual(t, webTag, depWeb.Spec.Template.Spec.Containers[0].Image)

	skaffold.Delete().InDir("examples/microservices").InNs(ns.Name).RunOrFail(t)
}

func TestDeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Deploy("--images", "index.docker.io/library/busybox:1").InDir("examples/kustomize").InNs(ns.Name).RunOrFail(t)

	dep := client.GetDeployment("kustomize-test")
	testutil.CheckDeepEqual(t, "index.docker.io/library/busybox:1", dep.Spec.Template.Spec.Containers[0].Image)

	skaffold.Delete().InDir("examples/kustomize").InNs(ns.Name).RunOrFail(t)
}

func TestDeployWithInCorrectConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	// We're not providing a tag for the getting-started image
	output, err := skaffold.Deploy().InDir("examples/getting-started").InNs(ns.Name).RunWithCombinedOutput(t)
	if err == nil {
		t.Errorf("expected to see an error since not every image tag is provided: %s", output)
	} else if !strings.Contains(string(output), "no tag provided for image [gcr.io/k8s-skaffold/skaffold-example]") {
		t.Errorf("failed without saying the reason: %s", output)
	}
}
