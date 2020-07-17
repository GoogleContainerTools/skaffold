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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

func TestPortForward(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	dir := "examples/microservices"
	skaffold.Run().InDir(dir).InNs(ns.Name).RunOrFail(t)

	cfg, err := kubectx.CurrentConfig()
	failNowIfError(t, err)

	kubectlCLI := kubectl.NewFromRunContext(&runcontext.RunContext{
		KubeContext: cfg.CurrentContext,
		Opts: config.SkaffoldOptions{
			Namespace: ns.Name,
		},
	})

	logrus.SetLevel(logrus.TraceLevel)
	portforward.SimulateDevCycle(t, kubectlCLI, ns.Name)
}

func TestRunPortForward(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Run("--port-forward", "--rpc-port", rpcAddr, "--enable-rpc").InDir("examples/microservices").InNs(ns.Name).RunBackground(t)

	_, entries := apiEvents(t, rpcAddr)

	address, localPort := getLocalPortFromPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name)
	assertResponseFromPort(t, address, localPort, constants.LeeroyAppResponse)
}

// TestDevPortForwardDeletePod tests that port forwarding works
// as expected. Then, the test force deletes a pod,
// and tests that the pod eventually comes up at the same port again.
func TestDevPortForwardDeletePod(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--port-forward", "--rpc-port", rpcAddr).InDir("examples/microservices").InNs(ns.Name).RunBackground(t)

	_, entries := apiEvents(t, rpcAddr)

	address, localPort := getLocalPortFromPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name)
	assertResponseFromPort(t, address, localPort, constants.LeeroyAppResponse)

	// now, delete all pods in this namespace.
	Run(t, ".", "kubectl", "delete", "pods", "--all", "-n", ns.Name)

	assertResponseFromPort(t, address, localPort, constants.LeeroyAppResponse)
}
