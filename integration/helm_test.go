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
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestHelmDeploy(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	ns, client := SetupNamespace(t)

	// To fix #1823, we make use of env variable templating for release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	skaffold.Deploy("--images", "us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold-helm").InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)

	dep := client.GetDeployment("skaffold-helm-" + ns.Name)
	testutil.CheckDeepEqual(t, dep.Name, dep.ObjectMeta.Labels["release"])

	skaffold.Delete().InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)
}

func TestDevHelmMultiConfig(t *testing.T) {
	var tests = []struct {
		description  string
		dir          string
		args         []string
		deployments  []string
		pods         []string
		env          []string
		targetLogOne string
		targetLogTwo string
	}{
		{
			description:  "helm-multi-config",
			dir:          "testdata/helm-multi-config/skaffold",
			deployments:  []string{"app1", "app2"},
			targetLogOne: "app1",
			targetLogTwo: "app2",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			if test.targetLogOne == "" || test.targetLogTwo == "" {
				t.SkipNow()
			}
			if test.dir == emptydir {
				err := os.MkdirAll(filepath.Join(test.dir, "emptydir"), 0755)
				t.Log("Creating empty directory")
				if err != nil {
					t.Errorf("Error creating empty dir: %s", err)
				}
			}

			ns, _ := SetupNamespace(t)

			out := skaffold.Run(test.args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunLive(t)
			defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).Run(t)

			WaitForLogs(t, out, test.targetLogOne)
			WaitForLogs(t, out, test.targetLogTwo)
		})
	}
}
