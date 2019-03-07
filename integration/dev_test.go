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

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDev(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	RunSkaffold(t, "build", "testdata/dev", "", "", nil)

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	cancel := make(chan bool)
	go RunSkaffoldNoFail(cancel, "dev", "testdata/dev", ns.Name, "", nil)
	defer func() { cancel <- true }()

	deployName := "test-dev"
	if err := kubernetesutil.WaitForDeploymentToStabilize(context.Background(), client, ns.Name, deployName, 10*time.Minute); err != nil {
		t.Fatalf("Timed out waiting for deployment to stabilize")
	}

	dep, err := client.AppsV1().Deployments(ns.Name).Get(deployName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatalf("Could not find dep: %s %s", ns.Name, deployName)
	}

	// Make a change to foo so that dev is forced to delete the Deployment and redeploy
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Make sure the old Deployment and the new Deployment are different
	err = wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
		newDep, err := client.AppsV1().Deployments(ns.Name).Get(deployName, meta_v1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return dep.GetGeneration() != newDep.GetGeneration(), nil
	})
	if err != nil {
		t.Fatalf("redeploy failed: %v", err)
	}
}
