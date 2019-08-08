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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
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

// TestPortForwardDeletePod tests that port forwarding works
// as expected. Then, the test force deletes a pod,
// and tests that the pod eventually comes up at the same port again.
func TestPortForwardDeletePod(t *testing.T) {
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

	em := portforward.NewEntryManager(os.Stdout, kubectlCLI)
	defer em.Stop()

	pfe := portforward.NewPortForwardEntry(em, latest.PortForwardResource{
		Type:      "deployment",
		Name:      "leeroy-web",
		Namespace: ns.Name,
		Port:      8080,
	})

	cleanup := portforward.OverridePortForwardEvent()
	defer cleanup()

	logrus.SetLevel(logrus.TraceLevel)

	// Start port forwarding
	portforward.ForwardPortForwardEntry(em, pfe)

	waitForResponseFromPort(t, pfe.LocalPort(), constants.LeeroyAppResponse)

	// now, delete all pods in this namespace.
	cmd := kubectlCLI.Command(context.Background(),
		"delete",
		"pods", "--all",
		"-n", ns.Name,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("error deleting all pods: %v \n %s", err, string(output))
	}
	// port forwarding should come up again on the same port
	waitForResponseFromPort(t, pfe.LocalPort(), constants.LeeroyAppResponse)
}

// waitForResponseFromPort waits for two minutes for the expected response at port.
func waitForResponseFromPort(t *testing.T, port int, expected string) {
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelTimeout()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for response from port %d", port)

		default:
			time.Sleep(1 * time.Second)
			resp, err := http.Get(fmt.Sprintf("http://%s:%d", util.Loopback, port))
			if err != nil {
				logrus.Infof("error getting response from port %d: %v", port, err)
				continue
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Infof("error reading response: %v", err)
				continue
			}
			if string(body) == expected {
				return
			}
			logrus.Infof("didn't get expected response from port. got: %s, expected: %s", string(body), expected)
		}
	}
}
