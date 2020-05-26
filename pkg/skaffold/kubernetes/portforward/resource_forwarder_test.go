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
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type testForwarder struct {
	forwardedResources forwardedResources
	forwardedPorts     util.PortSet
}

func (f *testForwarder) Forward(ctx context.Context, pfe *portForwardEntry) {
	f.forwardedResources.Store(pfe.key(), pfe)
	f.forwardedPorts.Set(pfe.localPort)
}

func (f *testForwarder) Monitor(*portForwardEntry, func()) {}

func (f *testForwarder) Terminate(pfe *portForwardEntry) {
	f.forwardedResources.Delete(pfe.key())
	f.forwardedPorts.Delete(pfe.localPort)
}

func newTestForwarder() *testForwarder {
	return &testForwarder{}
}

func mockRetrieveAvailablePort(_ string, taken map[int]struct{}, availablePorts []int) func(string, int, *util.PortSet) int {
	// Return first available port in ports that isn't taken
	var lock sync.Mutex
	return func(string, int, *util.PortSet) int {
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
			event.InitializeState(latest.Pipeline{}, "", true, true, true)
			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort("127.0.0.1", map[int]struct{}{}, test.availablePorts))
			t.Override(&retrieveServices, func(string, []string) ([]*latest.PortForwardResource, error) {
				return test.resources, nil
			})

			fakeForwarder := newTestForwarder()
			entryManager := NewEntryManager(ioutil.Discard, fakeForwarder)

			rf := NewResourceForwarder(entryManager, []string{"test"}, "", nil)
			if err := rf.Start(context.Background()); err != nil {
				t.Fatalf("error starting resource forwarder: %v", err)
			}

			// poll up to 10 seconds for the resources to be forwarded
			err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (bool, error) {
				return len(test.expected) == fakeForwarder.forwardedResources.Length(), nil
			})
			if err != nil {
				t.Fatalf("expected entries didn't match actual entries. Expected: \n %v Actual: \n %v", test.expected, fakeForwarder.forwardedResources.resources)
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
			expected:       newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false),
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
			expected: newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 9000, false),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort("127.0.0.1", map[int]struct{}{}, test.availablePorts))

			entryManager := NewEntryManager(ioutil.Discard, newTestForwarder())
			entryManager.forwardedResources = forwardedResources{
				resources: test.forwardedResources,
			}
			rf := NewResourceForwarder(entryManager, []string{"test"}, "", nil)
			actualEntry := rf.getCurrentEntry(test.resource)

			expectedEntry := test.expected
			expectedEntry.resource = test.resource
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
		event.InitializeState(latest.Pipeline{}, "", true, true, true)
		t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort("127.0.0.1", map[int]struct{}{}, []int{8080, 9000}))
		t.Override(&retrieveServices, func(string, []string) ([]*latest.PortForwardResource, error) {
			return []*latest.PortForwardResource{svc}, nil
		})

		fakeForwarder := newTestForwarder()
		entryManager := NewEntryManager(ioutil.Discard, fakeForwarder)

		rf := NewResourceForwarder(entryManager, []string{"test"}, "", []*latest.PortForwardResource{pod})
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

func mockClient(m kubernetes.Interface) func() (kubernetes.Interface, error) {
	return func() (kubernetes.Interface, error) {
		return m, nil
	}
}

func TestRetrieveServices(t *testing.T) {
	tests := []struct {
		description string
		namespaces  []string
		services    []*v1.Service
		expected    []*latest.PortForwardResource
	}{
		{
			description: "multiple services in multiple namespaces",
			namespaces:  []string{"test", "test1"},
			services: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc1",
						Namespace: "test",
						Labels: map[string]string{
							deploy.RunIDLabel: "9876-6789",
						},
					},
					Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 8080}}},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc2",
						Namespace: "test1",
						Labels: map[string]string{
							deploy.RunIDLabel: "9876-6789",
						},
					},
					Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 8081}}},
				},
			},
			expected: []*latest.PortForwardResource{{
				Type:      constants.Service,
				Name:      "svc1",
				Namespace: "test",
				Port:      8080,
				Address:   "127.0.0.1",
				LocalPort: 8080,
			}, {
				Type:      constants.Service,
				Name:      "svc2",
				Namespace: "test1",
				Port:      8081,
				Address:   "127.0.0.1",
				LocalPort: 8081,
			}},
		}, {
			description: "no services in given namespace",
			namespaces:  []string{"randon"},
			services: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc1",
						Namespace: "test",
						Labels: map[string]string{
							deploy.RunIDLabel: "9876-6789",
						},
					},
					Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 8080}}},
				},
			},
		}, {
			description: "services present but does not expose any port",
			namespaces:  []string{"test"},
			services: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc1",
						Namespace: "test",
						Labels: map[string]string{
							deploy.RunIDLabel: "9876-6789",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			objs := make([]runtime.Object, len(test.services))
			for i, s := range test.services {
				objs[i] = s
			}
			client := fakekubeclientset.NewSimpleClientset(objs...)
			t.Override(&kubernetesutil.Client, mockClient(client))

			actual, err := retrieveServiceResources(fmt.Sprintf("%s=9876-6789", deploy.RunIDLabel), test.namespaces)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
