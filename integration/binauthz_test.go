/*
Copyright 2023 The Skaffold Authors

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
	"os"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

const (
	binauthzClusterName = "integration-tests-binauthz"
)

// TestBinAuthZWithDeploy targets the integration-tests-binauthz cluster on k8s-skaffold which has '--binauthz-evaluation-mode=PROJECT_SINGLETON_POLICY_ENFORCE' set
func TestBinAuthZWithDeploy(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)
	if os.Getenv("GKE_CLUSTER_NAME") != binauthzClusterName {
		t.Skip("TestBinAuthZWithDeploy only runs against integration-tests-binauthz cluster")
	}

	ns, _ := SetupNamespace(t)
	expected := "Failed to create Pod for Deployment"

	// We're not providing a tag for the getting-started image
	output, err := skaffold.Deploy("--images", "index.docker.io/library/busybox:1", "--default-repo=").InDir("examples/kustomize").InNs(ns.Name).RunWithCombinedOutput(t)
	if err == nil {
		t.Errorf("expected to see an error since the image deployed to cluster should be denied based on binauthz policy: %s", output)
	} else if !strings.Contains(string(output), expected) {
		t.Errorf("unexpected error text - expected: %s, got: %s", expected, output)
	}
}
