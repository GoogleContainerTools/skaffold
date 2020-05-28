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
	"errors"
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func pod(name string) *v1.Pod {
	return &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func service(name string) *v1.Service {
	return &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

type anyPod struct{}

func (p *anyPod) Select(pod *v1.Pod) bool { return true }

type hasName struct {
	validNames []string
}

func (h *hasName) Select(pod *v1.Pod) bool {
	for _, validName := range h.validNames {
		if validName == pod.Name {
			return true
		}
	}
	return false
}

func TestPodWatcher(t *testing.T) {
	testutil.Run(t, "need to register first", func(t *testutil.T) {
		watcher := NewPodWatcher(&anyPod{}, []string{"ns"})
		cleanup, err := watcher.Start()
		defer cleanup()

		t.CheckErrorContains("no receiver was registered", err)
	})

	testutil.Run(t, "fail to get client", func(t *testutil.T) {
		t.Override(&Client, func() (kubernetes.Interface, error) { return nil, errors.New("unable to get client") })

		watcher := NewPodWatcher(&anyPod{}, []string{"ns"})
		watcher.Register(make(chan PodEvent))
		cleanup, err := watcher.Start()
		defer cleanup()

		t.CheckErrorContains("unable to get client", err)
	})

	testutil.Run(t, "fail to watch pods", func(t *testutil.T) {
		clientset := fake.NewSimpleClientset()
		t.Override(&Client, func() (kubernetes.Interface, error) { return clientset, nil })

		clientset.Fake.PrependWatchReactor("pods", func(action k8stesting.Action) (handled bool, ret watch.Interface, err error) {
			return true, nil, errors.New("unable to watch")
		})

		watcher := NewPodWatcher(&anyPod{}, []string{"ns"})
		watcher.Register(make(chan PodEvent))
		cleanup, err := watcher.Start()
		defer cleanup()

		t.CheckErrorContains("unable to watch", err)
	})

	testutil.Run(t, "filter 3 events", func(t *testutil.T) {
		clientset := fake.NewSimpleClientset()
		t.Override(&Client, func() (kubernetes.Interface, error) { return clientset, nil })

		podSelector := &hasName{
			validNames: []string{"pod1", "pod2", "pod3"},
		}
		events := make(chan PodEvent)
		watcher := NewPodWatcher(podSelector, []string{"ns1", "ns2"})
		watcher.Register(events)
		cleanup, err := watcher.Start()
		defer cleanup()
		t.CheckNoError(err)

		// Send three pod events among other events
		clientset.CoreV1().Pods("ns1").Create(pod("pod1"))
		clientset.CoreV1().Pods("ignored").Create(pod("ignored"))     // Different namespace
		clientset.CoreV1().Services("ns1").Create(service("ignored")) // Not a pod
		clientset.CoreV1().Pods("ns2").Create(pod("ignored"))         // Rejected by podSelector
		clientset.CoreV1().Pods("ns2").Create(pod("pod2"))
		clientset.CoreV1().Pods("ns2").Create(pod("pod3"))

		// Retrieve three events
		var podEvents []PodEvent
		podEvents = append(podEvents, <-events)
		podEvents = append(podEvents, <-events)
		podEvents = append(podEvents, <-events)
		close(events)

		// Order is not guaranteed since we watch multiple names concurrently.
		sort.Slice(podEvents, func(i, j int) bool { return podEvents[i].Pod.Name < podEvents[j].Pod.Name })
		t.CheckDeepEqual("pod1", podEvents[0].Pod.Name)
		t.CheckDeepEqual("pod2", podEvents[1].Pod.Name)
		t.CheckDeepEqual("pod3", podEvents[2].Pod.Name)
	})
}
