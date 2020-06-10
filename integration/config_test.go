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
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConfigListForContext(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	out := skaffold.Config("list", "-c", "testdata/config/config.yaml", "-k", "test-context").RunOrFailOutput(t)

	testutil.CheckContains(t, "default-repo: context-local-repository", string(out))
}

func TestConfigListForAll(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	out := skaffold.Config("list", "-c", "testdata/config/config.yaml", "--all").RunOrFailOutput(t)

	for _, output := range []string{
		"global:",
		"default-repo: global-repository",
		"kube-context: test-context",
		"default-repo: context-local-repository",
	} {
		testutil.CheckContains(t, output, string(out))
	}
}

func TestFailToSetUnrecognizedValue(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	err := skaffold.Config("set", "doubt-this-will-ever-be-a-config-key", "VALUE", "-c", "testdata/config/config.yaml", "--global").Run(t)

	testutil.CheckError(t, true, err)
}

func TestSetDefaultRepoForContext(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	file := testutil.TempFile(t, "config", nil)

	skaffold.Config("set", "default-repo", "REPO1", "-c", file, "-k", "test-context").RunOrFail(t)
	out := skaffold.Config("list", "-c", file, "-k", "test-context").RunOrFailOutput(t)

	testutil.CheckContains(t, "default-repo: REPO1", string(out))
}

func TestSetGlobalDefaultRepo(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	file := testutil.TempFile(t, "config", nil)

	skaffold.Config("set", "default-repo", "REPO2", "-c", file, "--global").RunOrFail(t)
	out := skaffold.Config("list", "-c", file, "--all").RunOrFailOutput(t)

	testutil.CheckContains(t, "default-repo: REPO2", string(out))
}
