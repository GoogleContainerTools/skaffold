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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

func TestPortForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	dir := "examples/microservices"
	skaffold.Run().InDir(dir).InNs(ns.Name).RunOrFailOutput(t)

	cfg, err := kubectx.CurrentConfig()
	if err != nil {
		t.Fatal(err)
	}

	kubectlCLI := kubectl.NewFromRunContext(&runcontext.RunContext{
		KubeContext: cfg.CurrentContext,
		Opts: config.SkaffoldOptions{
			Namespace: ns.Name,
		},
	})

	logrus.SetLevel(logrus.TraceLevel)
	portforward.WhiteBoxPortForwardCycle(t, kubectlCLI, ns.Name)
}
