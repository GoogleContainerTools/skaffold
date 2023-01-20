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
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestFixExclusiveOptions(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	out := skaffold.Fix().InDir("testdata/fix").RunOrFailOutput(t)
	out, err := skaffold.Fix("--overwrite", "--output=ignored").WithConfig("-").InDir("testdata/fix").
		WithStdin(out).RunWithCombinedOutput(t)
	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "cannot be used together", string(out))
}

func TestFixStdout(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	ns, _ := SetupNamespace(t)

	out := skaffold.Fix().InDir("testdata/fix").RunOrFailOutput(t)
	skaffold.Run().WithConfig("-").InDir("testdata/fix").InNs(ns.Name).WithStdin(out).RunOrFail(t)
}

func TestFixV2Beta29ToV3Alpha1PatchProfiles(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	out, err := skaffold.Fix().InDir("testdata/fix-v2beta29").RunWithCombinedOutput(t)
	testutil.CheckError(t, false, err)
	testutil.CheckContains(t, "/manifests/helm/releases/0/setValueTemplates/image.repository", string(out))
	testutil.CheckContains(t, "{{.IMAGE_REPO_skaffold_helm_v2}}", string(out))
	testutil.CheckContains(t, "/manifests/helm/releases/0/setValueTemplates/image.tag", string(out))
	testutil.CheckContains(t, "{{.IMAGE_TAG_skaffold_helm_v2}}@{{.IMAGE_DIGEST_skaffold_helm_v2}}", string(out))
}

func TestFixOutputFile(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	out := skaffold.Fix("--output", "updated.yaml").InDir("testdata/fix").RunOrFailOutput(t)
	testutil.CheckContains(t, "written to updated.yaml", string(out))
	defer os.Remove(filepath.Join("testdata", "fix", "updated.yaml"))

	f, err := os.ReadFile(filepath.Join("testdata", "fix", "updated.yaml"))
	testutil.CheckError(t, false, err)

	parsed := make(map[string]interface{})
	err = yaml.UnmarshalStrict(f, parsed)
	testutil.CheckError(t, false, err)
}
