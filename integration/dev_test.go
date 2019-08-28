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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDev(t *testing.T) {
	tests := []struct {
		description string
		trigger     string
	}{
		{
			description: "dev with polling trigger",
			trigger:     "polling",
		},
		{
			description: "dev with notify trigger",
			trigger:     "notify",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if testing.Short() {
				t.Skip("skipping integration test")
			}
			if ShouldRunGCPOnlyTests() {
				t.Skip("skipping test that is not gcp only")
			}

			Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
			defer Run(t, "testdata/dev", "rm", "foo")

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir("testdata/dev").RunOrFail(t)

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			stop := skaffold.Dev("--trigger", test.trigger).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)
			defer stop()

			dep := client.GetDeployment("test-dev")

			// Make a change to foo so that dev is forced to delete the Deployment and redeploy
			Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

			// Make sure the old Deployment and the new Deployment are different
			err := wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
				newDep := client.GetDeployment("test-dev")
				return dep.GetGeneration() != newDep.GetGeneration(), nil
			})
			testutil.CheckError(t, false, err)
		})
	}
}

func TestDevAPITriggers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	ns, k8sClient, deleteNs := SetupNamespace(t)
	defer deleteNs()

	rpcAddr := randomPort()

	stop := skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunBackground(t)
	defer stop()

	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	stream, err := readEventAPIStream(client, t, readRetries)
	if stream == nil {
		t.Fatalf("error retrieving event log: %v\n", err)
	}

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		stream.Recv()
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

	dep := k8sClient.GetDeployment("test-dev")

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Issue a build trigger
	client.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Build: true,
		},
	})

	// Ensure we see a build triggered in the event log
	err = wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetBuildEvent().GetArtifact() == "gcr.io/k8s-skaffold/test-dev", nil
	})
	testutil.CheckError(t, false, err)

	// Issue a deploy trigger
	client.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Deploy: true,
		},
	})

	// Ensure we see a deploy triggered in the event log
	err = wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetDeployEvent().GetStatus() == "In Progress", nil
	})
	testutil.CheckError(t, false, err)

	// Make sure the old Deployment and the new Deployment are different
	err = wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
		newDep := k8sClient.GetDeployment("test-dev")
		return dep.GetGeneration() != newDep.GetGeneration(), nil
	})
	testutil.CheckError(t, false, err)
}

func TestDevPortForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build("--cache-artifacts=true").InDir("examples/microservices").RunOrFail(t)

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	rpcAddr := randomPort()
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	cmd := skaffold.Dev("--port-forward", "--rpc-port", rpcAddr, "--cache-artifacts=true").InDir("examples/microservices").InNs(ns.Name).WithEnv(env)
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

	originalResponse := "leeroooooy app!!"
	replacementResponse := "test string"

	waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, originalResponse+"\n")

	original, perms, fErr := replaceInFile(originalResponse, replacementResponse, "examples/microservices/leeroy-app/app.go")
	if fErr != nil {
		t.Error(fErr)
	}
	defer func() {
		if original != nil {
			ioutil.WriteFile("examples/microservices/leeroy-app/app.go", original, perms)
		}
	}()

	waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, replacementResponse+"\n")
}

func TestDevPortForwardGKELoadBalancer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is gcp only")
	}

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/gke_loadbalancer").RunOrFail(t)

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	rpcAddr := randomPort()
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	cmd := skaffold.Dev("--port-forward", "--rpc-port", rpcAddr).InDir("testdata/gke_loadbalancer").InNs(ns.Name).WithEnv(env)
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

	waitForPortForwardEvent(t, entries, "gke-loadbalancer", "service", ns.Name, "hello!!\n")
}

func getLocalPortFromPortForwardEvent(t *testing.T, entries chan *proto.LogEntry, resourceName, resourceType, namespace string) int {
	timeout := time.After(1 * time.Minute)
	for {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for port forwarding event")
		case e := <-entries:
			switch e.Event.GetEventType().(type) {
			case *proto.Event_PortEvent:
				t.Logf("event received %v", e)
				if e.Event.GetPortEvent().ResourceName == resourceName &&
					e.Event.GetPortEvent().ResourceType == resourceType &&
					e.Event.GetPortEvent().Namespace == namespace {
					port := e.Event.GetPortEvent().LocalPort
					t.Logf("Detected %s/%s is forwarded to port %d", resourceType, resourceName, port)
					return int(port)
				}
			default:
				t.Logf("event received %v", e)
			}
		}
	}
}

func waitForPortForwardEvent(t *testing.T, entries chan *proto.LogEntry, resourceName, resourceType, namespace, expected string) {
	port := getLocalPortFromPortForwardEvent(t, entries, resourceName, resourceType, namespace)
	assertResponseFromPort(t, port, expected)
}

// assertResponseFromPort waits for two minutes for the expected response at port.
func assertResponseFromPort(t *testing.T, port int, expected string) {
	logrus.Infof("Waiting for response %s from port %d", expected, port)
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelTimeout()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for response from port %d", port)
		case <-time.After(1 * time.Second):
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

func replaceInFile(target, replacement, filepath string) ([]byte, os.FileMode, error) {
	fInfo, err := os.Stat(filepath)
	if err != nil {
		return nil, 0, err
	}
	original, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, 0, err
	}

	newContents := strings.Replace(string(original), target, replacement, -1)

	err = ioutil.WriteFile(filepath, []byte(newContents), 0)

	return original, fInfo.Mode(), err
}

func readEventAPIStream(client proto.SkaffoldServiceClient, t *testing.T, retries int) (proto.SkaffoldService_EventLogClient, error) {
	t.Helper()
	// read the event log stream from the skaffold grpc server
	var stream proto.SkaffoldService_EventLogClient
	var err error
	for i := 0; i < retries; i++ {
		stream, err = client.EventLog(context.Background())
		if err != nil {
			t.Logf("waiting for connection...")
			time.Sleep(waitTime)
			continue
		}
	}
	return stream, err
}

func TestDev_WithKubecontextOverride(t *testing.T) {
	testutil.Run(t, "skaffold run with kubecontext override", func(t *testutil.T) {
		if testing.Short() {
			t.Skip("skipping integration test")
		}

		dir := "examples/getting-started"
		pods := []string{"getting-started"}

		ns, client, deleteNs := SetupNamespace(t.T)
		defer deleteNs()

		modifiedKubeconfig, kubecontext, err := createModifiedKubeconfig(ns.Name)
		if err != nil {
			t.Fatal(err)
		}
		kubeconfig := t.NewTempDir().
			Write("kubeconfig", string(modifiedKubeconfig)).
			Path("kubeconfig")
		env := []string{fmt.Sprintf("KUBECONFIG=%s", kubeconfig)}

		// n.b. for the sake of this test the namespace must not be given explicitly
		skaffold.Run("--kube-context", kubecontext).InDir(dir).WithEnv(env).RunOrFail(t.T)

		client.WaitForPodsReady(pods...)

		// n.b. for the sake of this test the namespace must not be given explicitly
		skaffold.Delete("--kube-context", kubecontext).InDir(dir).WithEnv(env).RunOrFail(t.T)
	})
}

func createModifiedKubeconfig(namespace string) ([]byte, string, error) {
	// do not use context.CurrentConfig(), because it may have cached a different config
	kubeConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, "", err
	}

	contextName := "modified-context"
	if config.IsKindCluster(kubeConfig.CurrentContext) {
		contextName += "@kind"
	}

	activeContext := kubeConfig.Contexts[kubeConfig.CurrentContext]
	if activeContext == nil {
		return nil, "", fmt.Errorf("no active kube-context set")
	}
	// clear the namespace in the active context
	activeContext.Namespace = ""

	newContext := activeContext.DeepCopy()
	newContext.Namespace = namespace
	kubeConfig.Contexts[contextName] = newContext

	yaml, err := clientcmd.Write(*kubeConfig)
	return yaml, contextName, err
}
