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
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/google/go-cmp/cmp"
)

const (
	TestVersion = "vtest"
)

func TestHelmDeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !ShouldRunGCPOnlyTests() {
		t.Skip("skipping gcp only test")
	}

	helmDir := "testdata/helm"

	ns, client, deleteNs := SetupNamespace(t)
	// To fix #1823, we make use of env variable templating for release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	depName := fmt.Sprintf("skaffold-helm-%s", ns.Name)

	defer func() {
		skaffold.Delete().InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFail(t)
		deleteNs()
	}()

	runArgs := []string{"--images", "gcr.io/k8s-skaffold/skaffold-helm"}

	skaffold.Deploy(runArgs...).InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFailOutput(t)

	client.WaitForDeploymentsToStabilize(depName)

	expectedLabels := map[string]string{
		"app.kubernetes.io/managed-by": TestVersion,
		"release":                      depName,
		"skaffold.dev/deployer":        "helm",
	}

	// check if labels are set correctly for deploument
	dep := client.GetDeployment(depName)
	if d := diffLabels(expectedLabels, dep.ObjectMeta.Labels); d != "" {
		t.Errorf("did not find expected labels for dep %s: %s", depName, d)
	}
}

func diffLabels(expected map[string]string, actual map[string]string) string {
	extracted := map[string]string{}
	for k, v := range actual {
		switch k {
		case "app.kubernetes.io/managed-by":
			extracted[k] = TestVersion // ignore version since its runtime
		case "release":
			extracted[k] = v
		case "skaffold.dev/deployer":
			extracted[k] = v
		default:
			continue
		}
	}
	return cmp.Diff(extracted, expected)
}
