// +build integration

/*
Copyright 2018 Google LLC

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
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
)

func TestRunNoArgs(t *testing.T) {
	client, err := kubernetes.GetClientset()
	if err != nil {
		t.Fatalf("Test setup error: getting kubernetes client: %s", err)
	}
	defer func() {
		if err := client.CoreV1().Pods("default").Delete("skaffold", nil); err != nil {
			t.Fatalf("Error deleting pod %s", err)
		}
	}()
	cmd := exec.Command("skaffold", "run")
	cmd.Dir = "../examples/getting-started"
	out, outerr, err := util.RunCommand(cmd, nil)
	if err != nil {
		t.Fatalf("skaffold run: \nstdout: %s\nstderr: %s\nerror: %s", out, outerr, err)
	}

	if err := kubernetes.WaitForPodReady(client.CoreV1().Pods("default"), "skaffold"); err != nil {
		t.Fatalf("Timed out waiting for pod ready")
	}
}
