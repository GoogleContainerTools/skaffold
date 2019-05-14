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
	"strings"
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
	if os.Getenv("REMOTE_INTEGRATION") != "true" {
		t.Skip("skipping remote only test")
	}

	helmDir := "examples/helm-deployment"

	ns, client, deleteNs := SetupNamespace(t)
	// To fix #1823, we make use of env variable templating for release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	depName := fmt.Sprintf("skaffold-helm-%s", ns.Name)
	defer func() {
		skaffold.Delete().InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFail(t)
		deleteNs()
	}()

	skaffold.Deploy("--images", "gcr.io/k8s-skaffold/skaffold-helm").InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFailOutput(t)

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

	// check if atleast one pod has correct labels set.
	// Currently, we do a
	// 1. helm install or upgrade,
	// 2. grab the manifests for deployed resources,
	// 3. change the deployed manifests and apply.
	// In this example, we use a deployment. Applying labels after deployment triggers a spec change.
	// A new revision of deployments is rolled out https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#updating-a-deployment
	// The pod belonging to the earlier revision does not have the newly set labels correctly so.
	// There is no easy way to get Deployments -> Current ReplicaSet -> Pods.
	// Hence we have this hack to check if at the least one pod has all the right set of labels.
	// Alternatively we can `kubectl get  pods --namespace skaffoldtzbff -o=jsonpath='{range .items[*]}{.metadata.labels}{end}'`
	// and parse the output.
	foundPodWithCorrectLabels := false
	pods := client.GetPods()
	diffs := make([]string, len(pods.Items))
	for i, po := range pods.Items {
		diffs[i] = diffLabels(expectedLabels, po.ObjectMeta.Labels)
		if diffs[i] == "" {
			foundPodWithCorrectLabels = true
			break
		}
	}
	if !foundPodWithCorrectLabels {
		t.Errorf("Could not find one pod with all the labels. See diff:\n%s", strings.Join(diffs, "\n"))
	}

	skaffold.Delete().InDir(helmDir).InNs(ns.Name).WithEnv(env).RunOrFail(t)
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
