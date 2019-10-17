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
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewEntryManager(t *testing.T) {
	out := ioutil.Discard
	cli := &kubectl.CLI{}
	expected := EntryManager{
		output:             out,
		forwardedPorts:     newForwardedPorts(),
		forwardedResources: newForwardedResources(),
		EntryForwarder:     NewKubectlForwarder(out, cli),
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
	}, "", "", "", "", 9000, false)

	pfe2 := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      constants.Pod,
		Name:      "resource2",
		Namespace: "default",
	}, "", "", "", "", 9001, false)

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

func TestForwardedResources_ByResource(t *testing.T) {
	foo8080 := latest.PortForwardResource{
		Type:      latest.ResourceType("pod"),
		Name:      "foo",
		Namespace: "test",
		Port:      8080,
		LocalPort: 8080,
	}
	foo8081 := latest.PortForwardResource{
		Type:      latest.ResourceType("pod"),
		Name:      "foo",
		Namespace: "test",
		Port:      8081,
		LocalPort: 8081,
	}
	bar4500 := latest.PortForwardResource{
		Type:      latest.ResourceType("pod"),
		Name:      "bar",
		Namespace: "test",
		Port:      4500,
		LocalPort: 4500,
	}
	tests := []struct {
		description string
		pfes        []*portForwardEntry
		expected    []*portForwardEntry
	}{
		{
			description: "returns all ports forwarded for a foo resource",
			pfes: []*portForwardEntry{
				newPortForwardEntry(1, foo8080, "foo", "container1", "port1", "owner", 8080, true),
				newPortForwardEntry(1, foo8081, "foo", "container2", "port2", "owner", 8081, true),
				newPortForwardEntry(1, bar4500, "foo", "container1", "port", "owner", 4500, true),
			},
			expected: []*portForwardEntry{
				newPortForwardEntry(1, foo8080, "foo", "container1", "port1", "owner", 8080, true),
				newPortForwardEntry(1, foo8081, "foo", "container2", "port2", "owner", 8081, true),
			},
		},
		{
			description: "returns multiple entries for foo resource which belong to different resource version",
			pfes: []*portForwardEntry{
				newPortForwardEntry(1, foo8080, "foo", "container1", "port1", "owner", 8080, true),
				newPortForwardEntry(2, foo8081, "foo", "container2", "port2", "owner", 8081, true),
				newPortForwardEntry(1, bar4500, "foo", "container1", "port", "owner", 4500, true),
			},
			expected: []*portForwardEntry{
				newPortForwardEntry(1, foo8080, "foo", "container1", "port1", "owner", 8080, true),
				newPortForwardEntry(2, foo8081, "foo", "container2", "port2", "owner", 8081, true),
			},
		},
		{
			description: "matches none",
			pfes: []*portForwardEntry{
				newPortForwardEntry(1, bar4500, "foo", "container1", "port", "owner", 4500, true),
			},
			expected: []*portForwardEntry{},
		},
		{
			description: "returns empty when no port forwarded entries",
			pfes:        []*portForwardEntry{},
			expected:    []*portForwardEntry{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pf := newForwardedResources()
			for _, pfe := range test.pfes {
				pf.Store(pfe.key(), pfe)
			}
			actual := pf.ByResource(latest.ResourceType("pod"), "test", "foo")
			sort.SliceStable(actual, func(i, j int) bool { return actual[i].key() < actual[j].key() })
			sort.SliceStable(test.expected, func(i, j int) bool { return test.expected[i].key() < test.expected[j].key() })
			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expected, actual) {
				t.Errorf("Forwarded entries differs from expected entries. Expected: %v, Actual: %v", test.expected, actual)
			}
		})
	}
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

func TestResource_Equals(t *testing.T) {
	tests := []struct {
		description string
		r           resource
		pfe         *portForwardEntry
		expected    bool
	}{
		{
			description: "resource is equal",
			r:           newResource(latest.ResourceType("type1"), "test", "name"),
			pfe: &portForwardEntry{
				resourceVersion: 1,
				resource: latest.PortForwardResource{
					Type:      latest.ResourceType("type1"),
					Name:      "name",
					Namespace: "test",
					Port:      8080,
					LocalPort: 8080,
				},
				ownerReference: "dummy",
			},
			expected: true,
		},
		{
			description: "resource namespace is different",
			r:           newResource(latest.ResourceType("type1"), "test", "name"),
			pfe: &portForwardEntry{
				resourceVersion: 1,
				resource: latest.PortForwardResource{
					Type:      latest.ResourceType("type1"),
					Name:      "name",
					Namespace: "test-namespace",
					Port:      8080,
					LocalPort: 8080,
				},
				ownerReference: "dummy",
			},
		},
		{
			description: "resource name is different",
			r:           newResource(latest.ResourceType("type1"), "test", "foo"),
			pfe: &portForwardEntry{
				resourceVersion: 1,
				resource: latest.PortForwardResource{
					Type:      latest.ResourceType("type1"),
					Name:      "bar",
					Namespace: "test",
					Port:      8080,
					LocalPort: 8080,
				},
				ownerReference: "dummy",
			},
		},
		{
			description: "resource type is different",
			r:           newResource(latest.ResourceType("type1"), "test", "name"),
			pfe: &portForwardEntry{
				resourceVersion: 1,
				resource: latest.PortForwardResource{
					Type:      latest.ResourceType("pod"),
					Name:      "name",
					Namespace: "test",
					Port:      8080,
					LocalPort: 8080,
				},
				ownerReference: "dummy",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.r.Equals(test.pfe))
		})
	}
}
