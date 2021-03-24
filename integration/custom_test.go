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
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestCustomTest(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	config := "skaffold.yaml"
	expectedText := "bar\nbar\n"
	testDir := "testdata/custom-test"
	testFile := "testdata/custom-test/test"
	depFile := "testdata/custom-test/testdep"
	defer func() {
		os.Truncate(depFile, 0)
		os.Truncate(testFile, 0)
	}()

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir(testDir).WithConfig(config).RunOrFail(t)

	ns, client := SetupNamespace(t)

	skaffold.Dev().InDir(testDir).WithConfig(config).InNs(ns.Name).RunLive(t)

	client.WaitForPodsReady("custom-test-example")
	ioutil.WriteFile(depFile, []byte("foo"), 0644)

	err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		out, e := ioutil.ReadFile(testFile)
		failNowIfError(t, e)
		return string(out) == expectedText, nil
	})
	failNowIfError(t, err)
}
