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
	"os/exec"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDevSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Build().InDir("testdata/file-sync").InNs(ns.Name).RunOrFail(t)

	stop := skaffold.Dev().InDir("testdata/file-sync").InNs(ns.Name).RunBackground(t)
	defer stop()

	client.WaitForPodsReady("test-file-sync")

	Run(t, "testdata/file-sync", "mkdir", "-p", "test")
	Run(t, "testdata/file-sync", "touch", "test/foobar")
	defer Run(t, "testdata/file-sync", "rm", "-rf", "test")

	err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		_, err := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "ls", "/test").Output()
		return err == nil, nil
	})
	testutil.CheckError(t, false, err)
}
