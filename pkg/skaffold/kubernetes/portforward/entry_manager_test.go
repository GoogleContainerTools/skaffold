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

package portforward

import (
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func TestNewEntryManager(t *testing.T) {
	out := ioutil.Discard
	cli := &kubectl.CLI{}
	expected := EntryManager{
		output:             out,
		forwardedPorts:     newForwardedPorts(),
		forwardedResources: newForwardedResources(),
		EntryForwarder:     &KubectlForwarder{kubectl: cli, out: out},
	}
	actual := NewEntryManager(out, cli)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected result different from actual result. Expected: %v, Actual: %v", expected, actual)
	}
}

func TestStop(t *testing.T) {
	pfe1 := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      constants.Pod,
		Name:      "resource",
		Namespace: "default",
	}, "", "", "", 9000, false)

	pfe2 := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      constants.Pod,
		Name:      "resource2",
		Namespace: "default",
	}, "", "", "", 9001, false)

	em := NewEntryManager(ioutil.Discard, nil)

	em.forwardedResources = newForwardedResources()
	em.forwardedResources.Store("pod-resource-default-0", pfe1)
	em.forwardedResources.Store("pod-resource2-default-0", pfe2)

	em.forwardedPorts = newForwardedPorts()
	em.forwardedPorts.Store(9000, struct{}{})
	em.forwardedPorts.Store(9001, struct{}{})

	fakeForwarder := newTestForwarder()
	fakeForwarder.forwardedResources = em.forwardedResources
	em.EntryForwarder = fakeForwarder

	em.Stop()

	if count := fakeForwarder.forwardedResources.Length(); count != 0 {
		t.Fatalf("error stopping port forwarding. expected 0 entries and got %d", count)
	}

	if count := len(fakeForwarder.forwardedPorts.ports); count != 0 {
		t.Fatalf("error cleaning up ports. expected 0 entries and got %d", count)
	}
}

func TestForwardedPorts(t *testing.T) {
	pf := newForwardedPorts()

	// Try to store a port
	pf.Store(9000, struct{}{})

	// Try to load the port
	if _, ok := pf.LoadOrStore(9000, struct{}{}); !ok {
		t.Fatal("didn't load port 9000 correctly")
	}

	if _, ok := pf.LoadOrStore(4000, struct{}{}); ok {
		t.Fatal("didn't store port 4000 correctly")
	}

	if _, ok := pf.LoadOrStore(4000, struct{}{}); !ok {
		t.Fatal("didn't load port 4000 correctly")
	}

	// Try to store a non int, catch panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	pf.Store("not an int", struct{}{})
}

func TestForwardedResources(t *testing.T) {
	pf := newForwardedResources()

	// Try to store a resource
	pf.Store("resource", &portForwardEntry{})

	// Try to load the resource
	if _, ok := pf.Load("resource"); !ok {
		t.Fatal("didn't load resource correctly correctly")
	}

	// Try to load a resource that doesn't exist
	if actual, ok := pf.Load("dne"); ok || actual != nil {
		t.Fatal("loaded resource that doesn't exist")
	}

	// Try to store a string instead of *portForwardEntry
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	pf.Store("resource2", "not port forward entry")
}

//This is the same behavior
func TestGetAvailablePortOnForwardedPorts(t *testing.T) {
	AssertCompetingProcessesCanSucceed(newForwardedPorts(), t)
}

//TODO this is copy pasted to portforward.forwardedPorts testing as well - it should go away when we introduce port brokering
// https://github.com/GoogleContainerTools/skaffold/issues/2503
func AssertCompetingProcessesCanSucceed(ports util.ForwardedPorts, t *testing.T) {
	t.Helper()
	N := 100
	var (
		errors int32
		wg     sync.WaitGroup
	)
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {

			port := util.GetAvailablePort(4503, ports)

			l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", util.Loopback, port))
			if err != nil {
				atomic.AddInt32(&errors, 1)
			} else {
				l.Close()
			}
			time.Sleep(2 * time.Second)
			wg.Done()
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&errors) > 0 {
		t.Fatalf("A port that was available couldn't be used %d times", errors)
	}
}
