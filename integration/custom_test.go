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
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestCustomTest(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	config := "skaffold.yaml"

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/custom-test").WithConfig(config).RunOrFail(t)

	ns, client := SetupNamespace(t)

	skaffold.Dev().InDir("testdata/custom-test").WithConfig(config).InNs(ns.Name).RunBackground(t)

	client.WaitForPodsReady("custom-test-example")

	ioutil.WriteFile("testdata/custom-test/foo", []byte("foo"), 0644)
	defer func() { os.Truncate("testdata/custom-test/foo", 0) }()

	fileContent, err := ioutil.ReadFile("testdata/custom-test/foo")
	if err != nil || string(fileContent) != "bar" {
		t.Fatalf("Test failed. Existing file contents %s did not match expected %s", string(fileContent), "bar")
	}
	failNowIfError(t, err)
}
