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
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestAutomaticPortForwardPod(t *testing.T) {
	tests := []struct {
		description     string
		pods            []*v1.Pod
		forwarder       *testForwarder
		availablePorts  []int
		expectedPorts   []int
		expectedEntries map[string]*portForwardEntry
		shouldErr       bool
	}{
		{
			description:    "single container port",
			availablePorts: []int{8080},
			expectedPorts:  []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"owner-containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
						Address:   "127.0.0.1",
						LocalPort: 8080,
					},
					ownerReference:         "owner",
					automaticPodForwarding: true,
					portName:               "portname",
					localPort:              8080,
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
			description:    "unavailable container port",
			availablePorts: []int{9000},
			expectedPorts:  []int{9000},
			expectedEntries: map[string]*portForwardEntry{
				"owner-containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
						Address:   "127.0.0.1",
						LocalPort: 8080,
					},
					ownerReference:         "owner",
					automaticPodForwarding: true,
					containerName:          "containername",
					portName:               "portname",
					localPort:              9000,
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
			description:     "bad resource version",
			availablePorts:  []int{8080},
			expectedPorts:   nil,
			shouldErr:       true,
			expectedEntries: nil,
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
			description:    "two different container ports",
			availablePorts: []int{8080, 50051},
			expectedPorts:  []int{8080, 50051},
			expectedEntries: map[string]*portForwardEntry{
				"owner-containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
						Address:   "127.0.0.1",
						LocalPort: 8080,
					},
					ownerReference:         "owner",
					portName:               "portname",
					automaticPodForwarding: true,
					localPort:              8080,
				},
				"owner-containername2-namespace2-portname2-50051": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname2",
						Namespace: "namespace2",
						Port:      50051,
						Address:   "127.0.0.1",
						LocalPort: 50051,
					},
					ownerReference:         "owner",
					portName:               "portname2",
					automaticPodForwarding: true,
					localPort:              50051,
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
			description:    "two same container ports",
			availablePorts: []int{8080, 9000},
			expectedPorts:  []int{8080, 9000},
			expectedEntries: map[string]*portForwardEntry{
				"owner-containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					portName:        "portname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
						Address:   "127.0.0.1",
						LocalPort: 8080,
					},
					ownerReference:         "owner",
					automaticPodForwarding: true,
					localPort:              8080,
				},
				"owner-containername2-namespace2-portname2-8080": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					portName:        "portname2",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname2",
						Namespace: "namespace2",
						Port:      8080,
						Address:   "127.0.0.1",
						LocalPort: 8080,
					},
					ownerReference:         "owner",
					automaticPodForwarding: true,
					localPort:              9000,
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
			description:    "updated pod gets port forwarded",
			availablePorts: []int{8080},
			expectedPorts:  []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"owner-containername-namespace-portname-8080": {
					resourceVersion: 2,
					podName:         "podname",
					containerName:   "containername",
					portName:        "portname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
						Address:   "127.0.0.1",
						LocalPort: 8080,
					},
					ownerReference:         "owner",
					automaticPodForwarding: true,
					localPort:              8080,
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
		testutil.Run(t, test.description, func(t *testutil.T) {
			event.InitializeState(latest.Pipeline{}, "test", true, true, true)
			taken := map[int]struct{}{}
			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort("127.0.0.1", taken, test.availablePorts))
			t.Override(&topLevelOwnerKey, func(metav1.Object, string) string { return "owner" })

			if test.forwarder == nil {
				test.forwarder = newTestForwarder()
			}
			entryManager := NewEntryManager(ioutil.Discard, nil)
			entryManager.entryForwarder = test.forwarder

			p := NewWatchingPodForwarder(entryManager, kubernetes.NewImageList(), nil)
			for _, pod := range test.pods {
				err := p.portForwardPod(context.Background(), pod)
				t.CheckError(test.shouldErr, err)
			}

			t.CheckDeepEqual(test.expectedPorts, test.forwarder.forwardedPorts.List())

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			actualForwardedResources := test.forwarder.forwardedResources.resources
			if !reflect.DeepEqual(test.expectedEntries, actualForwardedResources) {
				t.Errorf("Forwarded entries differs from expected entries. Expected: %v, Actual: %v", test.expectedEntries, actualForwardedResources)
			}
		})
	}
}

func TestStartPodForwarder(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "default",
			ResourceVersion: "9",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "mycontainer",
				Image: "image",
				Ports: []v1.ContainerPort{{
					Name:          "myport",
					ContainerPort: 8080,
				}},
			}},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}

	tests := []struct {
		description   string
		entryExpected bool
		event         kubernetes.PodEvent
	}{
		{
			description:   "pod modified event",
			entryExpected: true,
			event: kubernetes.PodEvent{
				Type: watch.Modified,
				Pod:  pod,
			},
		},
		{
			description: "event is deleted",
			event: kubernetes.PodEvent{
				Type: watch.Deleted,
				Pod:  pod,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			event.InitializeState(latest.Pipeline{}, "", true, true, true)
			t.Override(&topLevelOwnerKey, func(metav1.Object, string) string { return "owner" })
			t.Override(&newPodWatcher, func(kubernetes.PodSelector, []string) kubernetes.PodWatcher {
				return &fakePodWatcher{
					events: []kubernetes.PodEvent{test.event},
				}
			})

			imageList := kubernetes.NewImageList()
			imageList.Add("image")

			fakeForwarder := newTestForwarder()
			entryManager := NewEntryManager(ioutil.Discard, fakeForwarder)

			p := NewWatchingPodForwarder(entryManager, imageList, nil)
			p.Start(context.Background())

			// wait for the pod resource to be forwarded
			err := wait.PollImmediate(10*time.Millisecond, 100*time.Millisecond, func() (bool, error) {
				_, ok := fakeForwarder.forwardedResources.Load("owner-mycontainer-default-myport-8080")
				return ok, nil
			})
			if err != nil && test.entryExpected {
				t.Fatalf("expected entry wasn't forwarded: %v", err)
			}
		})
	}
}

type fakePodWatcher struct {
	events   []kubernetes.PodEvent
	receiver chan<- kubernetes.PodEvent
}

func (f *fakePodWatcher) Register(receiver chan<- kubernetes.PodEvent) {
	f.receiver = receiver
}

func (f *fakePodWatcher) Start() (func(), error) {
	go func() {
		for _, event := range f.events {
			f.receiver <- event
		}
	}()

	return func() {}, nil
}
