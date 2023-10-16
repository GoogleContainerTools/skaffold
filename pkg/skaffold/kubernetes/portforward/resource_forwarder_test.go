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
	"io"
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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

type testForwarder struct {
	forwardedResources sync.Map
	forwardedPorts     util.PortSet
}

func (f *testForwarder) Forward(ctx context.Context, pfe *portForwardEntry) error {
	f.forwardedResources.Store(pfe.key(), pfe)
	f.forwardedPorts.Set(pfe.localPort)
	return nil
}

func (f *testForwarder) Monitor(*portForwardEntry, func()) {}

func (f *testForwarder) Terminate(pfe *portForwardEntry) {
	f.forwardedResources.Delete(pfe.key())
	f.forwardedPorts.Delete(pfe.localPort)
}

func (f *testForwarder) Start(io.Writer) {}

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
		Port:      schemautil.FromInt(8080),
	}

	svc2 := &latest.PortForwardResource{
		Type:      constants.Service,
		Name:      "svc2",
		Namespace: "default",
		Port:      schemautil.FromInt(9000),
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
			testEvent.InitializeState([]latest.Pipeline{{}})
			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(util.Loopback, map[int]struct{}{}, test.availablePorts))
			t.Override(&retrieveServices, func(context.Context, string, []string, string) ([]*latest.PortForwardResource, error) {
				return test.resources, nil
			})

			fakeForwarder := newTestForwarder()
			entryManager := NewEntryManager(fakeForwarder)

			rf := NewServicesForwarder(entryManager, "", "")
			if err := rf.Start(context.Background(), io.Discard, []string{"test"}); err != nil {
				t.Fatalf("error starting resource forwarder: %v", err)
			}

			// poll up to 10 seconds for the resources to be forwarded
			err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (bool, error) {
				return len(test.expected) == length(&fakeForwarder.forwardedResources), nil
			})
			if err != nil {
				t.Fatalf("expected entries didn't match actual entries.\nExpected: %v\n  Actual: %v", test.expected, print(&fakeForwarder.forwardedResources))
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
		expectedReq        int
		expected           *portForwardEntry
	}{
		{
			description: "port forward service",
			resource: latest.PortForwardResource{
				Type: "service",
				Name: "serviceName",
				Port: schemautil.FromInt(8080),
			},
			availablePorts: []int{8080},
			expectedReq:    8080,
			expected:       newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false),
		}, {
			description: "should not request system ports (1-1023)",
			resource: latest.PortForwardResource{
				Type: "service",
				Name: "serviceName",
				Port: schemautil.FromInt(80),
			},
			availablePorts: []int{8080},
			expectedReq:    0, // no local port requested as port 80 is a system port
			expected:       newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false),
		}, {
			description: "port forward existing deployment",
			resource: latest.PortForwardResource{
				Type:      "deployment",
				Namespace: "default",
				Name:      "depName",
				Port:      schemautil.FromInt(8080),
			},
			forwardedResources: map[string]*portForwardEntry{
				"deployment-depName-default-8080": {
					resource: latest.PortForwardResource{
						Type:      "deployment",
						Namespace: "default",
						Name:      "depName",
						Port:      schemautil.FromInt(8080),
					},
					localPort: 9000,
				},
			},
			expectedReq: -1, // retrieveAvailablePort should not be called as there is an assigned localPort
			expected:    newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 9000, false),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&retrieveAvailablePort, func(addr string, req int, ps *util.PortSet) int {
				t.CheckDeepEqual(test.expectedReq, req)
				return mockRetrieveAvailablePort(util.Loopback, map[int]struct{}{}, test.availablePorts)(addr, req, ps)
			})

			entryManager := NewEntryManager(newTestForwarder())
			entryManager.forwardedResources = sync.Map{}
			for k, v := range test.forwardedResources {
				entryManager.forwardedResources.Store(k, v)
			}
			rf := NewServicesForwarder(entryManager, "", "")
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
		Namespace: "test",
		Port:      schemautil.FromInt(8080),
	}

	tests := []struct {
		description       string
		userResources     []*latest.PortForwardResource
		namespaces        []string
		expectedResources []string
	}{
		{
			description: "pod should be found",
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod", Port: schemautil.FromInt(9000)},
			},
			namespaces: []string{"test"},
			expectedResources: []string{
				"pod-pod-test-9000",
			},
		},
		{
			description: "pod not available",
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod", Port: schemautil.FromInt(9000)},
			},
			namespaces:        []string{"test", "some"},
			expectedResources: []string{},
		},
		{
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod", Port: schemautil.FromInt(9000)},
				{Type: constants.Pod, Name: "pod", Namespace: "some", Port: schemautil.FromInt(9001)},
			},
			namespaces: []string{"test", "some"},
			expectedResources: []string{
				"pod-pod-some-9001",
			},
		},
		{
			description: "pod should be found with namespace with template",
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod", Namespace: "some-with-template-{{ .FOO }}", Port: schemautil.FromInt(9000)},
			},
			namespaces: []string{"test"},
			expectedResources: []string{
				"pod-pod-some-with-template-bar-9000",
			},
		},
		{
			description: "pod should be found with namespace with template",
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod", Namespace: "some-with-template-{{ .FOO }}", Port: schemautil.FromInt(9000)},
			},
			namespaces: []string{"test", "another"},
			expectedResources: []string{
				"pod-pod-some-with-template-bar-9000",
			},
		},
		{
			description: "pod should be found with name with template",
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod-{{ .FOO }}", Port: schemautil.FromInt(9000)},
			},
			namespaces: []string{"test"},
			expectedResources: []string{
				"pod-pod-bar-test-9000",
			},
		},
		{
			description: "pod should be found with name with template",
			userResources: []*latest.PortForwardResource{
				{Type: constants.Pod, Name: "pod-{{ .FOO }}", Namespace: "some-ns", Port: schemautil.FromInt(9000)},
			},
			namespaces: []string{"test", "another"},
			expectedResources: []string{
				"pod-pod-bar-some-ns-9000",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testEvent.InitializeState([]latest.Pipeline{{}})
			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(util.Loopback, map[int]struct{}{}, []int{8080, 9000}))
			t.Override(&retrieveServices, func(context.Context, string, []string, string) ([]*latest.PortForwardResource, error) {
				return []*latest.PortForwardResource{svc}, nil
			})

			fakeForwarder := newTestForwarder()
			entryManager := NewEntryManager(fakeForwarder)

			util.OSEnviron = func() []string {
				return []string{"FOO=bar"}
			}

			rf := NewUserDefinedForwarder(entryManager, "", test.userResources)
			if err := rf.Start(context.Background(), io.Discard, test.namespaces); err != nil {
				t.Fatalf("error starting resource forwarder: %v", err)
			}

			// poll up to 10 seconds for the resources to be forwarded
			err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (bool, error) {
				return len(test.expectedResources) == length(&fakeForwarder.forwardedResources), nil
			})
			for _, key := range test.expectedResources {
				pfe, found := fakeForwarder.forwardedResources.Load(key)
				t.CheckTrue(found)
				t.CheckNotNil(pfe)
			}
			if err != nil {
				t.Fatalf("expected entries didn't match actual entries.\nExpected: %v\n  Actual: %v", test.expectedResources, print(&fakeForwarder.forwardedResources))
			}
		})
	}
}

func mockClient(m kubernetes.Interface) func(string) (kubernetes.Interface, error) {
	return func(string) (kubernetes.Interface, error) {
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
							label.RunIDLabel: "9876-6789",
						},
					},
					Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 8080}}},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc2",
						Namespace: "test1",
						Labels: map[string]string{
							label.RunIDLabel: "9876-6789",
						},
					},
					Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 8081}}},
				},
			},
			expected: []*latest.PortForwardResource{{
				Type:      constants.Service,
				Name:      "svc1",
				Namespace: "test",
				Port:      schemautil.FromInt(8080),
				Address:   "127.0.0.1",
				LocalPort: 0,
			}, {
				Type:      constants.Service,
				Name:      "svc2",
				Namespace: "test1",
				Port:      schemautil.FromInt(8081),
				Address:   "127.0.0.1",
				LocalPort: 0,
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
							label.RunIDLabel: "9876-6789",
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
							label.RunIDLabel: "9876-6789",
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
			t.Override(&kubernetesclient.Client, mockClient(client))

			actual, err := retrieveServiceResources(context.Background(), fmt.Sprintf("%s=9876-6789", label.RunIDLabel), test.namespaces, "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
