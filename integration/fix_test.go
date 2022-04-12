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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFixExclusiveOptions(t *testing.T) {
	// TODO: Fix https://github.com/GoogleContainerTools/skaffold/issues/7033.
	t.Skipf("Fix https://github.com/GoogleContainerTools/skaffold/issues/7033")

	MarkIntegrationTest(t, CanRunWithoutGcp)

	out := skaffold.Fix().InDir("testdata/fix").RunOrFailOutput(t)
	out, err := skaffold.Fix("--overwrite", "--output=ignored").WithConfig("-").InDir("testdata/fix").
		WithStdin(out).RunWithCombinedOutput(t)
	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "cannot be used together", string(out))
}

func TestFixStdout(t *testing.T) {
	// TODO: Fix https://github.com/GoogleContainerTools/skaffold/issues/7033.
	t.Skipf("Fix https://github.com/GoogleContainerTools/skaffold/issues/7033")

	MarkIntegrationTest(t, CanRunWithoutGcp)
	ns, _ := SetupNamespace(t)

	out := skaffold.Fix().InDir("testdata/fix").RunOrFailOutput(t)
	skaffold.Run().WithConfig("-").InDir("testdata/fix").InNs(ns.Name).WithStdin(out).RunOrFail(t)
}

func TestFixOutputFile(t *testing.T) {
	// TODO: Fix https://github.com/GoogleContainerTools/skaffold/issues/7033.
	t.Skipf("Fix https://github.com/GoogleContainerTools/skaffold/issues/7033")

	MarkIntegrationTest(t, CanRunWithoutGcp)

	out := skaffold.Fix("--output", filepath.Join("updated.yaml")).InDir("testdata/fix").RunOrFailOutput(t)
	testutil.CheckContains(t, "written to updated.yaml", string(out))
	defer os.Remove(filepath.Join("testdata", "fix", "updated.yaml"))

	f, err := ioutil.ReadFile(filepath.Join("testdata", "fix", "updated.yaml"))
	testutil.CheckError(t, false, err)

	parsed := make(map[string]interface{})
	err = yaml.UnmarshalStrict(f, parsed)
	testutil.CheckError(t, false, err)
}
