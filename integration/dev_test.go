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
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	event "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	V2proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDevNotification(t *testing.T) {
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
			MarkIntegrationTest(t, CanRunWithoutGcp)
			Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
			defer Run(t, "testdata/dev", "rm", "foo")

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir("testdata/dev").RunOrFail(t)

			ns, client := SetupNamespace(t)

			rpcAddr := randomPort()
			skaffold.Dev("--rpc-port", rpcAddr, "--trigger", test.trigger).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

			dep := client.GetDeployment(testDev)

			_, entries := v2apiEvents(t, rpcAddr)

			// Wait for the first devloop to register target files to the monitor before running command to change target files
			failNowIfError(t, waitForV2Event(100*time.Second, entries, func(e *V2proto.Event) bool {
				taskEvent, ok := e.EventType.(*V2proto.Event_TaskEvent)
				return ok && taskEvent.TaskEvent.Task == string(constants.DevLoop) && taskEvent.TaskEvent.Status == event.Succeeded
			}))

			// Make a change to foo so that dev is forced to delete the Deployment and redeploy
			Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

			// Make sure the old Deployment and the new Deployment are different
			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				newDep := client.GetDeployment(testDev)
				t.Logf("old gen: %d, new gen: %d", dep.GetGeneration(), newDep.GetGeneration())
				return dep.GetGeneration() != newDep.GetGeneration(), nil
			})
			failNowIfError(t, err)
		})
	}
}

func TestDevGracefulCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("graceful cancel doesn't work on windows")
	}

	tests := []struct {
		name        string
		dir         string
		pods        []string
		deployments []string
	}{
		{
			name: "getting-started",
			dir:  "examples/getting-started",
			pods: []string{"getting-started"},
		},
		{
			name:        "multi-config-microservices",
			dir:         "examples/multi-config-microservices",
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
		{
			name: "multiple deployers",
			dir:  "testdata/deploy-multiple",
			pods: []string{"deploy-kubectl", "deploy-kustomize"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)

			ns, client := SetupNamespace(t)
			p, _ := skaffold.Dev("-vtrace").InDir(test.dir).InNs(ns.Name).StartWithProcess(t)
			client.WaitForPodsReady(test.pods...)
			client.WaitForDeploymentsToStabilize(test.deployments...)

			defer func() {
				state, _ := p.Wait()

				// We can't `recover()` from a remotely panicked process, but we can check exit code instead.
				// Exit code 2 means the process panicked.
				// https://github.com/golang/go/issues/24284
				if state.ExitCode() == 2 {
					t.Fail()
				}
			}()

			// once deployments are stable, send a SIGINT and make sure things cleanup correctly
			p.Signal(syscall.SIGINT)
		})
	}
}

func TestDevCancelWithDockerDeployer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("graceful cancel doesn't work on windows")
	}

	tests := []struct {
		description string
		dir         string
		containers  []string
	}{
		{
			description: "interrupt dev loop in Docker deployer",
			dir:         "testdata/docker-deploy",
			containers:  []string{"ernie", "bert"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			p, err := skaffold.Dev().InDir(test.dir).StartWithProcess(t)
			if err != nil {
				t.Fatalf("error starting skaffold dev process")
			}

			if err = waitForContainersRunning(t, test.containers...); err != nil {
				t.Fatalf("failed waiting for containers: %v", err)
			}

			p.Signal(syscall.SIGINT)

			state, _ := p.Wait()

			if state.ExitCode() != 0 {
				t.Fail()
			}
		})
	}
}

func TestDevAPIBuildTrigger(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	ns, _ := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)

	// Wait for the first devloop to register target files to the monitor before running command to change target files
	failNowIfError(t, waitForEvent(90*time.Second, entries, func(e *proto.LogEntry) bool {
		dle, ok := e.Event.EventType.(*proto.Event_DevLoopEvent)
		return ok && dle.DevLoopEvent.Status == event.Succeeded
	}))

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Issue a build trigger
	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Build: true,
		},
	})

	// Ensure we see a build triggered in the event log
	err := waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		return e.GetEvent().GetBuildEvent().GetArtifact() == testDev
	})
	failNowIfError(t, err)
}

func TestDevApiDeployTrigger(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)
	dep := client.GetDeployment(testDev)

	// Wait for the first devloop to register target files to the monitor before running command to change target files
	failNowIfError(t, waitForEvent(90*time.Second, entries, func(e *proto.LogEntry) bool {
		dle, ok := e.Event.EventType.(*proto.Event_DevLoopEvent)
		return ok && dle.DevLoopEvent.Status == event.Succeeded
	}))

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Issue a deploy trigger
	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Deploy: true,
		},
	})

	verifyDeployment(t, entries, client, dep)
}

func TestDevAPIAutoTriggers(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)
	dep := client.GetDeployment(testDev)

	// Wait for the first devloop to register target files to the monitor before running command to change target files
	failNowIfError(t, waitForEvent(90*time.Second, entries, func(e *proto.LogEntry) bool {
		dle, ok := e.Event.EventType.(*proto.Event_DevLoopEvent)
		return ok && dle.DevLoopEvent.Status == event.Succeeded
	}))

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
	err := waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		return e.GetEvent().GetBuildEvent().GetArtifact() == testDev
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
	err := waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		return e.GetEvent().GetDeployEvent().GetStatus() == InProgress
	})
	failNowIfError(t, err)

	// Make sure the old Deployment and the new Deployment are different
	err = wait.Poll(5*time.Second, 3*time.Minute, func() (bool, error) {
		newDep := client.GetDeployment(testDev)
		t.Logf("old gen: %d, new gen: %d", dep.GetGeneration(), newDep.GetGeneration())
		return dep.GetGeneration() != newDep.GetGeneration(), nil
	})
	failNowIfError(t, err)
}

func TestDevPortForward(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{
			name: "microservices",
			dir:  "examples/microservices"},
		{
			name: "multi-config-microservices",
			dir:  "examples/multi-config-microservices"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir(test.dir).RunOrFail(t)

			ns, _ := SetupNamespace(t)

			rpcAddr := randomPort()
			skaffold.Dev("--status-check=false", "--port-forward", "--rpc-port", rpcAddr).InDir(test.dir).InNs(ns.Name).RunBackground(t)

			_, entries := apiEvents(t, rpcAddr)

			waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, "leeroooooy app!!\n")

			original, perms, fErr := replaceInFile("leeroooooy app!!", "test string", fmt.Sprintf("%s/leeroy-app/app.go", test.dir))
			failNowIfError(t, fErr)
			defer func() {
				if original != nil {
					os.WriteFile(fmt.Sprintf("%s/leeroy-app/app.go", test.dir), original, perms)
				}
			}()

			waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, "test string\n")
		})
	}
}

func TestDevDeletePreviousBuiltImages(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{
			name: "microservices",
			dir:  "examples/microservices"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir(test.dir).RunOrFail(t)

			ns, k8sClient := SetupNamespace(t)

			rpcAddr := randomPort()
			skaffold.Dev("--status-check=false", "--port-forward", "--rpc-port", rpcAddr).InDir(test.dir).InNs(ns.Name).RunBackground(t)

			_, entries := apiEvents(t, rpcAddr)

			waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, "leeroooooy app!!\n")
			deployment := k8sClient.GetDeployment("leeroy-app")
			image := deployment.Spec.Template.Spec.Containers[0].Image

			original, perms, fErr := replaceInFile("leeroooooy app!!", "test string", fmt.Sprintf("%s/leeroy-app/app.go", test.dir))
			failNowIfError(t, fErr)
			defer func() {
				if original != nil {
					os.WriteFile(fmt.Sprintf("%s/leeroy-app/app.go", test.dir), original, perms)
				}
			}()

			waitForPortForwardEvent(t, entries, "leeroy-app", "service", ns.Name, "test string\n")
			client := SetupDockerClient(t)
			ctx := context.TODO()
			wait.Poll(3*time.Second, time.Minute*2, func() (done bool, err error) {
				return !client.ImageExists(ctx, image), nil
			})
		})
	}
}

func TestDevPortForwardDefaultNamespace(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("examples/microservices").RunOrFail(t)

	rpcAddr := randomPort()
	skaffold.Dev("--status-check=false", "--port-forward", "--rpc-port", rpcAddr).InDir("examples/microservices").RunBackground(t)
	defer skaffold.Delete().InDir("examples/microservices").Run(t)
	_, entries := apiEvents(t, rpcAddr)

	// No namespace was provided to `skaffold dev`, so we assume "default"
	waitForPortForwardEvent(t, entries, "leeroy-app", "service", "default", "leeroooooy app!!\n")

	original, perms, fErr := replaceInFile("leeroooooy app!!", "test string", "examples/microservices/leeroy-app/app.go")
	failNowIfError(t, fErr)
	defer func() {
		if original != nil {
			os.WriteFile("examples/microservices/leeroy-app/app.go", original, perms)
		}
	}()

	waitForPortForwardEvent(t, entries, "leeroy-app", "service", "default", "test string\n")
}

func TestDevPortForwardGKELoadBalancer(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)
	t.Skip("Skipping until resolved")

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
	timeout := time.After(2 * time.Minute)
	for {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for port forwarding event")
		case e := <-entries:
			switch e.Event.GetEventType().(type) {
			case *proto.Event_PortEvent:
				t.Logf("port event received: %v", e)
				if e.Event.GetPortEvent().ResourceName == resourceName &&
					e.Event.GetPortEvent().ResourceType == resourceType &&
					e.Event.GetPortEvent().Namespace == namespace {
					address := e.Event.GetPortEvent().Address
					port := e.Event.GetPortEvent().LocalPort
					t.Logf("Detected %s/%s is forwarded to address %s port %d", resourceType, resourceName, address, port)
					return address, int(port)
				}
			default:
				t.Logf("event received: %v", e)
			}
		}
	}
}

//nolint:unparam
func waitForPortForwardEvent(t *testing.T, entries chan *proto.LogEntry, resourceName, resourceType, namespace, expected string) {
	address, port := getLocalPortFromPortForwardEvent(t, entries, resourceName, resourceType, namespace)
	assertResponseFromPort(t, address, port, expected)
}

// assertResponseFromPort waits for two minutes for the expected response at port.
func assertResponseFromPort(t *testing.T, address string, port int, expected string) {
	url := fmt.Sprintf("http://%s:%d", address, port)
	t.Logf("Waiting on %s to return: %s", url, expected)
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Minute)
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
			body, err := io.ReadAll(resp.Body)
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
	original, err := os.ReadFile(filepath)
	if err != nil {
		return nil, 0, err
	}

	newContents := strings.ReplaceAll(string(original), target, replacement)

	err = os.WriteFile(filepath, []byte(newContents), 0)

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
		skaffold.Run("--kube-context", kubecontext).InDir("examples/getting-started").WithEnv(env).InNs(ns.Name).RunOrFail(t.T)

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
	if config.IsK3dCluster(kubeConfig.CurrentContext) {
		contextName = "k3d-" + contextName
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
