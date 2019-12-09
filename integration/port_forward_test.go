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
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/proto"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

func TestPortForward(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	dir := "examples/microservices"
	skaffold.Run().InDir(dir).InNs(ns.Name).RunOrFail(t)

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

// TestPortForwardDeletePod tests that port forwarding works
// as expected. Then, the test force deletes a pod,
// and tests that the pod eventually comes up at the same port again.
func TestPortForwardDeletePod(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	rpcAddr := randomPort()
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	cmd := skaffold.Dev("--port-forward", "--rpc-port", rpcAddr, "-v=info").InDir("examples/microservices").InNs(ns.Name).WithEnv(env)
	stop := cmd.RunBackground(t)
	defer stop()

	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	// create a grpc connection. Increase number of reties for helm.
	stream, err := readEventAPIStream(client, t, 20)
	if stream == nil {
		t.Fatalf("error retrieving event log: %v\n", err)
	}

	// read entries from the log
	entries := make(chan *proto.LogEntry)
	go func() {
		for {
			entry, _ := stream.Recv()
			if entry != nil {
				entries <- entry
			}
		}
	}()

	address, localPort := getLocalPortFromPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name)
	assertResponseFromPort(t, address, localPort, constants.LeeroyAppResponse)

	// now, delete all pods in this namespace.
	logrus.Infof("Deleting all pods in namespace %s", ns.Name)
	kubectlCLI := getKubectlCLI(t, ns.Name)
	killPodsCmd := kubectlCLI.Command(context.Background(),
		"delete",
		"pods", "--all",
		"-n", ns.Name,
	)

	if output, err := killPodsCmd.CombinedOutput(); err != nil {
		t.Fatalf("error deleting all pods: %v \n %s", err, string(output))
	}
	// port forwarding should come up again on the same port
	assertResponseFromPort(t, address, localPort, constants.LeeroyAppResponse)
}

func getKubectlCLI(t *testing.T, ns string) *kubectl.CLI {
	cfg, err := kubectx.CurrentConfig()
	if err != nil {
		t.Fatal(err)
	}

	return kubectl.NewFromRunContext(&runcontext.RunContext{
		KubeContext: cfg.CurrentContext,
		Opts: config.SkaffoldOptions{
			Namespace: ns,
		},
	})
}
