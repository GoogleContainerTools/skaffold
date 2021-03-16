/*
Copyright 2021 The Skaffold Authors

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
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestCustomTest(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	config := "skaffold.yaml"
	expectedText := "bar"
	testDir := "testdata/custom-test"
	testFile := "testdata/custom-test/foo"

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir(testDir).WithConfig(config).RunOrFail(t)

	ns, _ := SetupNamespace(t)

	skaffold.Dev().InDir(testDir).WithConfig(config).InNs(ns.Name).RunBackground(t)

	ioutil.WriteFile(testFile, []byte("foo"), 0644)
	defer func() { os.Truncate(testFile, 0) }()

	found := false
	for start := time.Now(); time.Since(start) < time.Second*5; {
		fileContent, err := ioutil.ReadFile(testFile)
		if err != nil {
			failNowIfError(t, err)
		}
		actualText := strings.TrimSuffix(string(fileContent), "\n")
		found = actualText == expectedText
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("Test failed. File contents did not match with expected.")
	}
}
