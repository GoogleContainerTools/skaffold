// +build integration

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
	"os/exec"
	"testing"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDevSync(t *testing.T) {
	ns, deleteNs := SetupNamespace(t)
	defer deleteNs()

	RunSkaffold(t, "build", "examples/test-file-sync", ns.Name, "", nil)

	cancel := make(chan bool)
	go RunSkaffoldNoFail(cancel, "dev", "examples/test-file-sync", ns.Name, "", nil)
	defer func() { cancel <- true }()

	if err := kubernetesutil.WaitForPodReady(context.Background(), Client.CoreV1().Pods(ns.Name), "test-file-sync"); err != nil {
		t.Fatalf("Timed out waiting for pod ready")
	}

	Run(t, "examples/test-file-sync", "mkdir", "-p", "test")
	Run(t, "examples/test-file-sync", "touch", "test/foobar")
	defer Run(t, "examples/test-file-sync", "rm", "-rf", "test")

	err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		cmd := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "ls", "/test")
		_, err := util.RunCmdOut(cmd)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("checking if /test dir exists in container: %v", err)
	}
}
