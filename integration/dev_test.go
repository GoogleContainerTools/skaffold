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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDevNotification(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

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
			Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
			defer Run(t, "testdata/dev", "rm", "foo")

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir("testdata/dev").RunOrFail(t)

			ns, client := SetupNamespace(t)

			skaffold.Dev("--trigger", test.trigger).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

			dep := client.GetDeployment("test-dev")

			// Make a change to foo so that dev is forced to delete the Deployment and redeploy
			Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

			// Make sure the old Deployment and the new Deployment are different
			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				newDep := client.GetDeployment("test-dev")
				logrus.Infof("old gen: %d, new gen: %d", dep.GetGeneration(), newDep.GetGeneration())
				return dep.GetGeneration() != newDep.GetGeneration(), nil
			})
			failNowIfError(t, err)
		})
	}
}

func TestDevAPITriggers(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		<-entries
	}

	dep := client.GetDeployment("test-dev")

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Issue a build trigger
	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Build: true,
		},
	})

	// Ensure we see a build triggered in the event log
	err := wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetBuildEvent().GetArtifact() == "test-dev", nil
	})
	failNowIfError(t, err)

	// Issue a deploy trigger
	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Deploy: true,
		},
	})

	verifyDeployment(t, entries, client, dep)
}

func TestDevAPIAutoTriggers(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		<-entries
	}

	dep := client.GetDeployment("test-dev")

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Enable auto build
	rpcClient.AutoBuild(context.Background(), &proto.TriggerRequest{
		State: &proto.TriggerState{
			Val: &proto.TriggerState_Enabled{
				Enabled: true,
			},
		},
	})
	// Ensure we see a build triggered in the event log
	err := wait.Poll(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetBuildEvent().GetArtifact() == "test-dev", nil
	})
	failNowIfError(t, err)

	rpcClient.AutoDeploy(context.Background(), &proto.TriggerRequest{
		State: &proto.TriggerState{
			Val: &proto.TriggerState_Enabled{
				Enabled: true,
			},
		},
	})
	verifyDeployment(t, entries, client, dep)
}

func verifyDeployment(t *testing.T, entries chan *proto.LogEntry, client *NSKubernetesClient, dep *appsv1.Deployment) {
	// Ensure we see a deploy triggered in the event log
	err := wait.Poll(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetDeployEvent().GetStatus() == "In Progress", nil
	})
	failNowIfError(t, err)

	// Make sure the old Deployment and the new Deployment are different
	err = wait.Poll(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		newDep := client.GetDeployment("test-dev")
		logrus.Infof("old gen: %d, new gen: %d", dep.GetGeneration(), newDep.GetGeneration())
		return dep.GetGeneration() != newDep.GetGeneration(), nil
	})
	failNowIfError(t, err)
}

func TestDevPortForward(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("examples/microservices").RunOrFail(t)

	ns, _ := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--status-check=false", "--port-forward", "--rpc-port", rpcAddr).InDir("examples/microservices").InNs(ns.Name).RunBackground(t)

	_, entries := apiEvents(t, rpcAddr)

	waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, "leeroooooy app!!\n")

	original, perms, fErr := replaceInFile("leeroooooy app!!", "test string", "examples/microservices/leeroy-app/app.go")
	failNowIfError(t, fErr)
	defer func() {
		if original != nil {
			ioutil.WriteFile("examples/microservices/leeroy-app/app.go", original, perms)
		}
	}()

	waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, "test string\n")
}

func TestDevPortForwardGKELoadBalancer(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/gke_loadbalancer").RunOrFail(t)

	ns, _ := SetupNamespace(t)

	rpcAddr := randomPort()
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	skaffold.Dev("--port-forward", "--rpc-port", rpcAddr).InDir("testdata/gke_loadbalancer").InNs(ns.Name).WithEnv(env).RunBackground(t)

	_, entries := apiEvents(t, rpcAddr)

	waitForPortForwardEvent(t, entries, "gke-loadbalancer", "service", ns.Name, "hello!!\n")
}

func getLocalPortFromPortForwardEvent(t *testing.T, entries chan *proto.LogEntry, resourceName, resourceType, namespace string) (string, int) {
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
					address := e.Event.GetPortEvent().Address
					port := e.Event.GetPortEvent().LocalPort
					t.Logf("Detected %s/%s is forwarded to address %s port %d", resourceType, resourceName, address, port)
					return address, int(port)
				}
			default:
				t.Logf("event received %v", e)
			}
		}
	}
}

func waitForPortForwardEvent(t *testing.T, entries chan *proto.LogEntry, resourceName, resourceType, namespace, expected string) {
	address, port := getLocalPortFromPortForwardEvent(t, entries, resourceName, resourceType, namespace)
	assertResponseFromPort(t, address, port, expected)
}

// assertResponseFromPort waits for two minutes for the expected response at port.
func assertResponseFromPort(t *testing.T, address string, port int, expected string) {
	url := fmt.Sprintf("http://%s:%d", address, port)
	t.Logf("Waiting on %s to return: %s", url, expected)
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelTimeout()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for response from port %d", port)
		case <-time.After(1 * time.Second):
			client := http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				t.Logf("[retriable error]: %v", err)
				continue
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Logf("[retriable error] reading response: %v", err)
				continue
			}
			if string(body) == expected {
				return
			}
			t.Logf("[retriable error] didn't get expected response from port. got: %s, expected: %s", string(body), expected)
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

func TestDev_WithKubecontextOverride(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	testutil.Run(t, "skaffold run with kubecontext override", func(t *testutil.T) {
		ns, client := SetupNamespace(t.T)

		modifiedKubeconfig, kubecontext, err := createModifiedKubeconfig(ns.Name)
		failNowIfError(t, err)

		kubeconfig := t.NewTempDir().
			Write("kubeconfig", string(modifiedKubeconfig)).
			Path("kubeconfig")
		env := []string{fmt.Sprintf("KUBECONFIG=%s", kubeconfig)}

		// n.b. for the sake of this test the namespace must not be given explicitly
		skaffold.Run("--kube-context", kubecontext).InDir("examples/getting-started").WithEnv(env).RunOrFail(t.T)

		client.WaitForPodsReady("getting-started")
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
		contextName = "kind-" + contextName
	}

	if kubeConfig.CurrentContext == constants.DefaultMinikubeContext {
		contextName = constants.DefaultMinikubeContext // skip, since integration test with minikube runs on single cluster
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
