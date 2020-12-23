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
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestHelmDeploy(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	ns, client := SetupNamespace(t)

	// To fix #1823, we make use of env variable templating for release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	skaffold.Deploy("--images", "gcr.io/k8s-skaffold/skaffold-helm").InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)

	dep := client.GetDeployment("skaffold-helm-" + ns.Name)
	testutil.CheckDeepEqual(t, dep.Name, dep.ObjectMeta.Labels["release"])

	skaffold.Delete().InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)
}
