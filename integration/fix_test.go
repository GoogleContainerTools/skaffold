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
	"bytes"
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func TestFix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	fixCmd := exec.Command("skaffold", "fix", "-f", "skaffold.yaml")
	fixCmd.Dir = "testdata/fix"
	out, err := util.RunCmdOut(fixCmd)
	if err != nil {
		t.Fatalf("skaffold fix: %v", err)
	}

	runCmd := exec.Command("skaffold", "run", "--namespace", ns.Name, "-f", "-")
	runCmd.Dir = "testdata/fix"
	runCmd.Stdin = bytes.NewReader(out)

	if out, err := util.RunCmdOut(runCmd); err != nil {
		t.Fatalf("skaffold run: %v, %s", err, out)
	}
}
