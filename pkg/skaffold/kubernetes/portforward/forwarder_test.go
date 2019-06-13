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

func TestNewBaseForwarder(t *testing.T) {
	out := ioutil.Discard
	namespaces := []string{"ns1", "ns2"}
	expected := BaseForwarder{
		output:             out,
		namespaces:         namespaces,
		forwardedPorts:     &sync.Map{},
		forwardedResources: &sync.Map{},
		EntryForwarder:     &KubectlForwarder{},
	}
	actual := NewBaseForwarder(out, namespaces)
	// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected differs from actual. Expected: %v, Actual: %v", expected, actual)
	}
}

func TestStop(t *testing.T) {
	pfe1 := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      constants.PodResourceType,
			Name:      "resource",
			Namespace: "default",
		},
	}
	pfe2 := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      constants.PodResourceType,
			Name:      "resource2",
			Namespace: "default",
		},
	}

	bf := NewBaseForwarder(ioutil.Discard, nil)

	bf.forwardedResources = &sync.Map{}
	bf.forwardedResources.Store("pod-resource-default-0", pfe1)
	bf.forwardedResources.Store("pod-resource2-default-0", pfe2)

	fakeForwarder := newTestForwarder(nil)
	fakeForwarder.forwardedEntries = bf.forwardedResources
	bf.EntryForwarder = fakeForwarder

	bf.Stop()

	if count := lengthSyncMap(fakeForwarder.forwardedEntries); count != 0 {
		t.Fatalf("error stopping port forwarding. expected 0 entries and got %d", count)
	}
}
