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
	"context"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Deploy("--images", "index.docker.io/library/busybox:1").InDir("examples/kustomize").InNs(ns.Name).RunOrFail(t)

	depName := "kustomize-test"
	if err := kubernetesutil.WaitForDeploymentToStabilize(context.Background(), client, ns.Name, depName, 10*time.Minute); err != nil {
		t.Fatalf("Timed out waiting for deployment to stabilize")
	}

	dep, err := client.AppsV1().Deployments(ns.Name).Get(depName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatalf("Could not find deployment: %s %s", ns.Name, depName)
	}

	if dep.Spec.Template.Spec.Containers[0].Image != "index.docker.io/library/busybox:1" {
		t.Fatalf("Wrong image name in kustomized deployment: %s", dep.Spec.Template.Spec.Containers[0].Image)
	}

	skaffold.Delete().InDir("examples/kustomize").InNs(ns.Name).RunOrFail(t)
}
