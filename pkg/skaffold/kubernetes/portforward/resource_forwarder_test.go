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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/wait"
)

type testForwarder struct {
	forwardedEntries *sync.Map
	forwardedPorts   *sync.Map

	forwardErr error
}

func (f *testForwarder) Forward(ctx context.Context, pfe *portForwardEntry) error {
	f.forwardedEntries.Store(pfe.key(), pfe)
	f.forwardedPorts.Store(pfe.localPort, true)
	return f.forwardErr
}

func (f *testForwarder) Terminate(pfe *portForwardEntry) {
	f.forwardedEntries.Delete(pfe.key())
	f.forwardedPorts.Delete(pfe.resource.Port)
}

func newTestForwarder(forwardErr error) *testForwarder {
	return &testForwarder{
		forwardedEntries: &sync.Map{},
		forwardedPorts:   &sync.Map{},
		forwardErr:       forwardErr,
	}
}

func mockRetrieveAvailablePort(taken map[int]struct{}, availablePorts []int) func(int, *sync.Map) int {
	// Return first available port in ports that isn't taken
	lock := sync.Mutex{}
	return func(int, *sync.Map) int {
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
	svc1 := latest.PortForwardResource{
		Type:      constants.Service,
		Name:      "svc1",
		Namespace: "default",
		Port:      8080,
	}

	svc2 := latest.PortForwardResource{
		Type:      constants.Service,
		Name:      "svc2",
		Namespace: "default",
		Port:      9000,
	}

	tests := []struct {
		description    string
		resources      []latest.PortForwardResource
		availablePorts []int
		expected       map[string]*portForwardEntry
	}{
		{
			description:    "forward two services",
			resources:      []latest.PortForwardResource{svc1, svc2},
			availablePorts: []int{8080, 9000},
			expected: map[string]*portForwardEntry{
				"service-svc1-default-8080": {
					resource:  svc1,
					localPort: 8080,
				},
				"service-svc2-default-9000": {
					resource:  svc2,
					localPort: 9000,
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			event.InitializeState(&runcontext.RunContext{Cfg: &latest.Pipeline{Build: latest.BuildConfig{}}})
			fakeForwarder := newTestForwarder(nil)
			rf := NewResourceForwarder(NewEntryManager(ioutil.Discard), "")
			rf.EntryForwarder = fakeForwarder

			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(map[int]struct{}{}, test.availablePorts))
			t.Override(&retrieveServices, func(string) ([]latest.PortForwardResource, error) {
				return test.resources, nil
			})

			if err := rf.Start(context.Background()); err != nil {
				t.Fatalf("error starting resource forwarder: %v", err)
			}
			// poll up to 10 seconds for the resources to be forwarded
			err := wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
				return len(test.expected) == lengthSyncMap(fakeForwarder.forwardedEntries), nil
			})
			if err != nil {
				t.Fatalf("expected entries didn't match actual entries. Expected: \n %v Actual: \n %v", test.expected, fakeForwarder.forwardedEntries)
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
				localPort: 8080,
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
				localPort: 9000,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			expectedEntry := test.expected
			expectedEntry.resource = test.resource

			rf := NewResourceForwarder(NewEntryManager(ioutil.Discard), "")
			rf.forwardedResources = generateSyncMap(test.forwardedResources)

			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(map[int]struct{}{}, test.availablePorts))

			actualEntry := rf.getCurrentEntry(test.resource)
			t.CheckDeepEqual(expectedEntry, actualEntry, cmp.AllowUnexported(portForwardEntry{}))
		})
	}
}

func generateSyncMap(m map[string]*portForwardEntry) *sync.Map {
	sm := &sync.Map{}
	for k, v := range m {
		sm.Store(k, v)
	}
	return sm
}

func lengthSyncMap(m *sync.Map) int {
	count := 0
	m.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
