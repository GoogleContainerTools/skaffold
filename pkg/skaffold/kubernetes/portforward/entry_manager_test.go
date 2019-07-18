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
	"io/ioutil"
	"reflect"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func newForwardedPorts() forwardedPorts {
	return forwardedPorts{
		lock:  &sync.Mutex{},
		ports: map[int]struct{}{},
	}
}

func newForwardedResources() forwardedResources {
	return forwardedResources{
		lock:      &sync.Mutex{},
		resources: map[string]*portForwardEntry{},
	}
}

func TestNewEntryManager(t *testing.T) {
	out := ioutil.Discard
	expected := EntryManager{
		output:             out,
		forwardedPorts:     newForwardedPorts(),
		forwardedResources: newForwardedResources(),
		EntryForwarder:     &KubectlForwarder{},
	}
	actual := NewEntryManager(out)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected result different from actual result. Expected: %v, Actual: %v", expected, actual)
	}
}

func TestStop(t *testing.T) {
	pfe1 := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      constants.Pod,
			Name:      "resource",
			Namespace: "default",
		},
		localPort: 9000,
	}
	pfe2 := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      constants.Pod,
			Name:      "resource2",
			Namespace: "default",
		},
		localPort: 9001,
	}

	em := NewEntryManager(ioutil.Discard)

	em.forwardedResources = newForwardedResources()
	em.forwardedResources.Store("pod-resource-default-0", pfe1)
	em.forwardedResources.Store("pod-resource2-default-0", pfe2)

	em.forwardedPorts = newForwardedPorts()
	em.forwardedPorts.Store(9000, struct{}{})
	em.forwardedPorts.Store(9001, struct{}{})

	fakeForwarder := newTestForwarder(nil)
	fakeForwarder.forwardedResources = em.forwardedResources
	em.EntryForwarder = fakeForwarder

	em.Stop()

	if count := len(fakeForwarder.forwardedResources.resources); count != 0 {
		t.Fatalf("error stopping port forwarding. expected 0 entries and got %d", count)
	}

	if count := len(fakeForwarder.forwardedPorts.ports); count != 0 {
		t.Fatalf("error cleaning up ports. expected 0 entries and got %d", count)
	}
}
