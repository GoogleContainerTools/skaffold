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
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type testForwarder struct {
	forwardedResources forwardedResources
	forwardedPorts     forwardedPorts
}

func (f *testForwarder) Forward(ctx context.Context, pfe *portForwardEntry) {
	f.forwardedResources.Store(pfe.key(), pfe)
	f.forwardedPorts.Store(pfe.localPort, true)
}

func (f *testForwarder) Monitor(_ *portForwardEntry, _ func()) {}

func (f *testForwarder) Terminate(pfe *portForwardEntry) {
	f.forwardedResources.Delete(pfe.key())
	f.forwardedPorts.Delete(pfe.resource.Port)
}

func newTestForwarder() *testForwarder {
	return &testForwarder{
		forwardedResources: newForwardedResources(),
		forwardedPorts:     newForwardedPorts(),
	}
}

func mockRetrieveAvailablePort(taken map[int]struct{}, availablePorts []int) func(int, util.ForwardedPorts) int {
	// Return first available port in ports that isn't taken
	lock := sync.Mutex{}
	return func(int, util.ForwardedPorts) int {
		for _, p := range availablePorts {
			lock.Lock()
			if _, ok := taken[p]; ok {
				lock.Unlock()
				continue
			}
			taken[p] = struct{}{}
			lock.Unlock()
			return p
		}
		return -1
	}
}

func TestStart(t *testing.T) {
	svc1 := &latest.PortForwardResource{
		Type:      constants.Service,
		Name:      "svc1",
		Namespace: "default",
		Port:      8080,
	}

	svc2 := &latest.PortForwardResource{
		Type:      constants.Service,
		Name:      "svc2",
		Namespace: "default",
		Port:      9000,
	}

	tests := []struct {
		description    string
		resources      []*latest.PortForwardResource
		availablePorts []int
		expected       map[string]*portForwardEntry
	}{
		{
			description:    "forward two services",
			resources:      []*latest.PortForwardResource{svc1, svc2},
			availablePorts: []int{8080, 9000},
			expected: map[string]*portForwardEntry{
				"service-svc1-default-8080": {
					resource:  *svc1,
					localPort: 8080,
				},
				"service-svc2-default-9000": {
					resource:  *svc2,
					localPort: 9000,
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			event.InitializeState(latest.BuildConfig{})
			fakeForwarder := newTestForwarder()
			rf := NewResourceForwarder(NewEntryManager(ioutil.Discard, nil), "", nil)
			rf.EntryForwarder = fakeForwarder

			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(map[int]struct{}{}, test.availablePorts))
			t.Override(&retrieveServices, func(string) ([]*latest.PortForwardResource, error) {
				return test.resources, nil
			})

			if err := rf.Start(context.Background()); err != nil {
				t.Fatalf("error starting resource forwarder: %v", err)
			}
			// poll up to 10 seconds for the resources to be forwarded
			err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (bool, error) {
				return len(test.expected) == fakeForwarder.forwardedResources.Length(), nil
			})
			if err != nil {
				t.Fatalf("expected entries didn't match actual entries. Expected: \n %v Actual: \n %v", test.expected, fakeForwarder.forwardedResources)
			}
		})
	}
}

func TestGetCurrentEntryFunc(t *testing.T) {
	tests := []struct {
		description        string
		forwardedResources map[string]*portForwardEntry
		availablePorts     []int
		resource           latest.PortForwardResource
		expected           *portForwardEntry
	}{
		{
			description: "port forward service",
			resource: latest.PortForwardResource{
				Type: "service",
				Name: "serviceName",
				Port: 8080,
			},
			availablePorts: []int{8080},
			expected: &portForwardEntry{
				localPort:       8080,
				terminationLock: &sync.Mutex{},
			},
		}, {
			description: "port forward existing deployment",
			resource: latest.PortForwardResource{
				Type:      "deployment",
				Namespace: "default",
				Name:      "depName",
				Port:      8080,
			},
			forwardedResources: map[string]*portForwardEntry{
				"deployment-depName-default-8080": {
					resource: latest.PortForwardResource{
						Type:      "deployment",
						Namespace: "default",
						Name:      "depName",
						Port:      8080,
					},
					localPort: 9000,
				},
			},
			expected: &portForwardEntry{
				localPort:       9000,
				terminationLock: &sync.Mutex{},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			expectedEntry := test.expected
			expectedEntry.resource = test.resource

			rf := NewResourceForwarder(NewEntryManager(ioutil.Discard, nil), "", nil)
			rf.forwardedResources = forwardedResources{
				resources: test.forwardedResources,
				lock:      &sync.Mutex{},
			}

			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(map[int]struct{}{}, test.availablePorts))

			actualEntry := rf.getCurrentEntry(test.resource)
			t.CheckDeepEqual(expectedEntry, actualEntry, cmp.AllowUnexported(portForwardEntry{}, sync.Mutex{}))
		})
	}
}

func TestUserDefinedResources(t *testing.T) {
	svc := &latest.PortForwardResource{
		Type:      constants.Service,
		Name:      "svc1",
		Namespace: "default",
		Port:      8080,
	}

	pod := &latest.PortForwardResource{
		Type:      constants.Pod,
		Name:      "pod",
		Namespace: "default",
		Port:      9000,
	}

	expected := map[string]*portForwardEntry{
		"service-svc1-default-8080": {
			resource:  *svc,
			localPort: 8080,
		},
		"pod-pod-default-9000": {
			resource:  *pod,
			localPort: 9000,
		},
	}

	testutil.Run(t, "one service and one user defined pod", func(t *testutil.T) {
		event.InitializeState(latest.BuildConfig{})
		fakeForwarder := newTestForwarder()
		rf := NewResourceForwarder(NewEntryManager(ioutil.Discard, nil), "", []*latest.PortForwardResource{pod})
		rf.EntryForwarder = fakeForwarder

		t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(map[int]struct{}{}, []int{8080, 9000}))
		t.Override(&retrieveServices, func(string) ([]*latest.PortForwardResource, error) {
			return []*latest.PortForwardResource{svc}, nil
		})

		if err := rf.Start(context.Background()); err != nil {
			t.Fatalf("error starting resource forwarder: %v", err)
		}
		// poll up to 10 seconds for the resources to be forwarded
		err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (bool, error) {
			return len(expected) == fakeForwarder.forwardedResources.Length(), nil
		})
		if err != nil {
			t.Fatalf("expected entries didn't match actual entries. Expected: \n %v Actual: \n %v", expected, fakeForwarder.forwardedResources.resources)
		}
	})
}
