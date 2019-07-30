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

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/apimachinery/pkg/util/wait"
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

	stop := skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)
	defer stop()

	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	stream, err := readEventAPIStream(client, t)
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
	skaffold.Build().InDir("examples/microservices").RunOrFail(t)

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	stop := skaffold.Dev("--port-forward").InDir("examples/microservices").InNs(ns.Name).RunBackground(t)
	defer stop()

	err := wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
		resp, err := http.Get("http://localhost:50053")
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}
		return "leeroooooy app!!\n" == string(body), nil
	})
	testutil.CheckError(t, false, err)

	original, perms, fErr := replaceInFile("leeroooooy app!!", "test string", "examples/microservices/leeroy-app/app.go")
	if fErr != nil {
		t.Error(fErr)
	}
	defer func() {
		if original != nil {
			ioutil.WriteFile("examples/microservices/leeroy-app/app.go", original, perms)
		}
	}()

	err = wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
		resp, err := http.Get("http://localhost:50053")
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}
		return "test string\n" == string(body), nil
	})

	testutil.CheckError(t, false, err)
}

func TestDevPortForwardGKELoadBalancer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/gke_loadbalancer").RunOrFail(t)

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	rpcAddr := randomPort()
	cmd := skaffold.Dev("--port-forward", "--rpc-port", rpcAddr).InDir("testdata/gke_loadbalancer").InNs(ns.Name)
	stop := cmd.RunBackground(t)
	defer stop()

	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	// create a grpc connection
	stream, err := readEventAPIStream(client, t)
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

	body := []byte{}
	err = wait.PollImmediate(time.Millisecond*500, 5*time.Minute, func() (bool, error) {
		e := <-entries
		switch e.Event.GetEventType().(type) {
		case *proto.Event_PortEvent:
			if e.Event.GetPortEvent().ResourceName == "gke-loadbalancer" &&
				e.Event.GetPortEvent().ResourceType == "service" {
				port := e.Event.GetPortEvent().LocalPort
				t.Logf("Detected service/gke-loadbalancer is forwarded to port %d", port)
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
				if err != nil {
					t.Errorf("could not get service/gke-loadbalancer due to %s", err)
				}
				defer resp.Body.Close()
				body, err = ioutil.ReadAll(resp.Body)
				return true, err
			}
			return false, nil
		default:
			return false, nil
		}
	})

	testutil.CheckErrorAndDeepEqual(t, false, err, string(body), "hello!!\n")
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

func readEventAPIStream(client proto.SkaffoldServiceClient, t *testing.T) (proto.SkaffoldService_EventLogClient, error) {
	t.Helper()
	// read the event log stream from the skaffold grpc server
	var stream proto.SkaffoldService_EventLogClient
	var err error
	for i := 0; i < readRetries; i++ {
		stream, err = client.EventLog(context.Background())
		if err != nil {
			t.Logf("waiting for connection...")
			time.Sleep(waitTime)
			continue
		}
	}
	return stream, err
}
