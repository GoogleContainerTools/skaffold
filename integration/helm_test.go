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
	"fmt"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/google/go-cmp/cmp"
)

func TestHelmDeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if os.Getenv("REMOTE_INTEGRATION") != "true" {
		t.Skip("skipping remote only test")
	}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	// To fix #1823, we make use of templating release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	depName := fmt.Sprintf("skaffold-helm-%s", ns.Name)
	helmDir := "examples/helm-deployment"
	skaffold.Deploy().InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFailOutput(t)

	client.WaitForDeploymentsToStabilize(depName)

	expectedLabels := map[string]string{
		"app.kubernetes.io/managed-by": "true",
		"release":                      depName,
		"skaffold.dev/deployer":        "helm",
	}
	//check if labels are set correctly for deploument
	dep := client.GetDeployment(depName)
	extractLabels(t, fmt.Sprintf("Dep: %s", depName), expectedLabels, dep.ObjectMeta.Labels)

	//check if labels are set correctly for pods
	pods := client.GetPods()
	// FIX ME the first pod dies.
	fmt.Println(pods.Items[1])
	po := pods.Items[0]
	extractLabels(t, fmt.Sprintf("Pod:%s", po.ObjectMeta.Name), expectedLabels, po.ObjectMeta.Labels)

	skaffold.Delete().InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFail(t)
}

func extractLabels(t *testing.T, name string, expected map[string]string, actual map[string]string) {
	extracted := map[string]string{}
	for k, v := range actual {
		switch k {
		case "app.kubernetes.io/managed-by":
			extracted[k] = "true" // ignore version since its runtime
		case "release":
			extracted[k] = v
		case "skaffold.dev/deployer":
			extracted[k] = v
		default:
			continue
		}
	}
	if d := cmp.Diff(extracted, expected); d != "" {
		t.Errorf("expected to see %s labels for %s. Diff %s", expected, name, d)
	}
}
