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

func TestRunTest(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description  string
		testDir      string
		testFile     string
		args         []string
		skipTests    bool
		expectedText string
	}{
		{
			description:  "Run test",
			testDir:      "testdata/run-test",
			testFile:     "testdata/run-test/test",
			skipTests:    false,
			expectedText: "foo\n",
		},
		{
			description:  "Run test with skip test false",
			testDir:      "testdata/run-test",
			testFile:     "testdata/run-test/test",
			args:         []string{"--skip-tests=false"},
			skipTests:    false,
			expectedText: "foo\n",
		},
		{
			description: "Run test with skip test true",
			testDir:     "testdata/run-test",
			testFile:    "testdata/run-test/test",
			args:        []string{"--skip-tests=True"},
			skipTests:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func() {
				os.Remove(test.testFile)
			}()

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir(test.testDir).RunOrFail(t)

			ns, client := SetupNamespace(t)
			skaffold.Run(test.args...).InDir(test.testDir).InNs(ns.Name).RunLive(t)

			client.WaitForPodsReady("run-test")

			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				_, e := os.Stat(test.testFile)
				if test.skipTests {
					if !os.IsNotExist(e) {
						t.Fatalf("Tests are not skipped.")
					}
					return true, nil
				}
				out, e := ioutil.ReadFile(test.testFile)
				failNowIfError(t, e)
				return string(out) == test.expectedText, nil
			})
			failNowIfError(t, err)
		})
	}
}
