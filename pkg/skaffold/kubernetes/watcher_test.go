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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func pod(name string) *v1.Pod {
	return &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func TestAggregatePodWatcher(t *testing.T) {
	testutil.Run(t, "fail to get client", func(t *testutil.T) {
		t.Override(&Client, func() (kubernetes.Interface, error) { return nil, errors.New("unable to get client") })

		cleanup, err := AggregatePodWatcher([]string{"ns"}, nil)
		defer cleanup()

		t.CheckErrorContains("unable to get client", err)
	})

	testutil.Run(t, "fail to watch pods", func(t *testutil.T) {
		clientset := fake.NewSimpleClientset()
		t.Override(&Client, func() (kubernetes.Interface, error) { return clientset, nil })

		clientset.Fake.PrependWatchReactor("pods", func(action k8stesting.Action) (handled bool, ret watch.Interface, err error) {
			return true, nil, errors.New("unable to watch")
		})

		cleanup, err := AggregatePodWatcher([]string{"ns"}, nil)
		defer cleanup()

		t.CheckErrorContains("unable to watch", err)
	})

	testutil.Run(t, "watch 3 events", func(t *testutil.T) {
		clientset := fake.NewSimpleClientset()
		t.Override(&Client, func() (kubernetes.Interface, error) { return clientset, nil })

		events := make(chan watch.Event)
		cleanup, err := AggregatePodWatcher([]string{"ns1", "ns2"}, events)
		defer cleanup()
		t.CheckNoError(err)

		// Send three events
		clientset.CoreV1().Pods("ns1").Create(pod("pod1"))
		clientset.CoreV1().Pods("ns2").Create(pod("pod2"))
		clientset.CoreV1().Pods("ns2").Create(pod("pod3"))

		// Retrieve three events
		<-events
		<-events
		<-events
	})
}
