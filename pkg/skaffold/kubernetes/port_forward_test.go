/*
Copyright 2018 The Skaffold Authors

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
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testForwarder struct {
	forwardedEntries map[string]*portForwardEntry
	forwardedPorts   map[int32]bool

	forwardErr error
	stopErr    error
}

func (f *testForwarder) Forward(pfe *portForwardEntry) error {
	f.forwardedEntries[pfe.key()] = pfe
	f.forwardedPorts[pfe.port] = true
	return f.forwardErr
}

func (f *testForwarder) Stop(pfe *portForwardEntry) error {
	delete(f.forwardedEntries, pfe.key())
	delete(f.forwardedPorts, pfe.port)
	return f.stopErr
}

func newTestForwarder(forwardErr, stopErr error) *testForwarder {
	return &testForwarder{
		forwardedEntries: map[string]*portForwardEntry{},
		forwardedPorts:   map[int32]bool{},
		forwardErr:       forwardErr,
		stopErr:          stopErr,
	}
}

func TestPortForwardPod(t *testing.T) {
	var tests = []struct {
		description     string
		pods            []*v1.Pod
		forwarder       *testForwarder
		expectedPorts   map[int32]bool
		expectedEntries map[string]*portForwardEntry
		shouldErr       bool
	}{
		{
			description: "single container port",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			expectedEntries: map[string]*portForwardEntry{
				"containername-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					port:            8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "10000000000a",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
			forwarder: newTestForwarder(fmt.Errorf(""), nil),
			shouldErr: true,
			expectedEntries: map[string]*portForwardEntry{
				"containername-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					port:            8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
			expectedEntries: map[string]*portForwardEntry{
				"containername-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					port:            8080,
				},
				"containername2-50051": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					port:            50051,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername2",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 50051,
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
			},
			expectedEntries: map[string]*portForwardEntry{
				"containername-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					port:            8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername2",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
			expectedEntries: map[string]*portForwardEntry{
				"containername-8080": {
					resourceVersion: 2,
					podName:         "podname",
					containerName:   "containername",
					port:            8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
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
			p := NewPortForwarder(ioutil.Discard, NewImageList())
			if test.forwarder == nil {
				test.forwarder = newTestForwarder(nil, nil)
			}
			p.Forwarder = test.forwarder

			for _, pod := range test.pods {
				err := p.portForwardPod(pod)
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
