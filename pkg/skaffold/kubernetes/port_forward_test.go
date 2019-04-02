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

package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testForwarder struct {
	forwardedEntries map[string]*portForwardEntry
	forwardedPorts   map[int32]bool

	forwardErr error
}

func (f *testForwarder) Forward(ctx context.Context, pfe *portForwardEntry) error {
	f.forwardedEntries[pfe.key()] = pfe
	f.forwardedPorts[pfe.localPort] = true
	return f.forwardErr
}

func (f *testForwarder) Terminate(pfe *portForwardEntry) {
	delete(f.forwardedEntries, pfe.key())
	delete(f.forwardedPorts, pfe.port)
}

func mockRetrieveAvailablePort(taken map[int]struct{}, availablePorts []int) func(int, *sync.Map) int {
	// Return first available port in ports that isn't taken
	return func(int, *sync.Map) int {
		for _, p := range availablePorts {
			if _, ok := taken[p]; ok {
				continue
			}
			taken[p] = struct{}{}
			return p
		}
		return -1
	}
}

func newTestForwarder(forwardErr error) *testForwarder {
	return &testForwarder{
		forwardedEntries: map[string]*portForwardEntry{},
		forwardedPorts:   map[int32]bool{},
		forwardErr:       forwardErr,
	}
}

func TestPortForwardPod(t *testing.T) {
	var tests = []struct {
		description     string
		pods            []*v1.Pod
		forwarder       *testForwarder
		expectedPorts   map[int32]bool
		expectedEntries map[string]*portForwardEntry
		availablePorts  []int
		shouldErr       bool
	}{
		{
			description: "single container port",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			availablePorts: []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"containername-podname-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					namespace:       "namespace",
					portName:        "portname",
					port:            8080,
					localPort:       8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "unavailable container port",
			expectedPorts: map[int32]bool{
				9000: true,
			},
			expectedEntries: map[string]*portForwardEntry{
				"containername-podname-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					namespace:       "namespace",
					portName:        "portname",
					port:            8080,
					localPort:       9000,
				},
			},
			availablePorts: []int{9000},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description:     "bad resource version",
			expectedPorts:   map[int32]bool{},
			shouldErr:       true,
			expectedEntries: map[string]*portForwardEntry{},
			availablePorts:  []int{8080},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "10000000000a",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "forward error",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			forwarder:      newTestForwarder(fmt.Errorf("")),
			shouldErr:      true,
			availablePorts: []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"containername-podname-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					namespace:       "namespace",
					portName:        "portname",
					port:            8080,
					localPort:       8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "two different container ports",
			expectedPorts: map[int32]bool{
				8080:  true,
				50051: true,
			},
			availablePorts: []int{8080, 50051},
			expectedEntries: map[string]*portForwardEntry{
				"containername-podname-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					namespace:       "namespace",
					portName:        "portname",
					port:            8080,
					localPort:       8080,
				},
				"containername2-podname2-namespace2-portname2-50051": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					namespace:       "namespace2",
					portName:        "portname2",
					port:            50051,
					localPort:       50051,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname2",
						ResourceVersion: "1",
						Namespace:       "namespace2",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername2",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 50051,
										Name:          "portname2",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "two same container ports",
			expectedPorts: map[int32]bool{
				8080: true,
				9000: true,
			},
			availablePorts: []int{8080, 9000},
			expectedEntries: map[string]*portForwardEntry{
				"containername-podname-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					namespace:       "namespace",
					portName:        "portname",
					port:            8080,
					localPort:       8080,
				},
				"containername2-podname2-namespace2-portname2-8080": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					namespace:       "namespace2",
					portName:        "portname2",
					port:            8080,
					localPort:       9000,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname2",
						ResourceVersion: "1",
						Namespace:       "namespace2",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername2",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname2",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "updated pod gets port forwarded",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			availablePorts: []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"containername-podname-namespace-portname-8080": {
					resourceVersion: 2,
					podName:         "podname",
					containerName:   "containername",
					namespace:       "namespace",
					portName:        "portname",
					port:            8080,
					localPort:       8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "2",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			taken := map[int]struct{}{}
			originalGetAvailablePort := util.GetAvailablePort
			retrieveAvailablePort = mockRetrieveAvailablePort(taken, test.availablePorts)
			defer func() {
				retrieveAvailablePort = originalGetAvailablePort
			}()

			p := NewPortForwarder(ioutil.Discard, &TailLabelSelector{}, []string{""})
			if test.forwarder == nil {
				test.forwarder = newTestForwarder(nil)
			}
			p.Forwarder = test.forwarder

			for _, pod := range test.pods {
				err := p.portForwardPod(context.Background(), pod)
				testutil.CheckError(t, test.shouldErr, err)
			}

			// Error is already checked above
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedPorts, test.forwarder.forwardedPorts)

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expectedEntries, test.forwarder.forwardedEntries) {
				t.Errorf("Forwarded entries differs from expected entries. Expected: %s, Actual: %s", test.expectedEntries, test.forwarder.forwardedEntries)
			}
		})
	}
}
