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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestNewEntryManager(t *testing.T) {
	out := ioutil.Discard
	expected := EntryManager{
		output:             out,
		forwardedPorts:     &sync.Map{},
		forwardedResources: &sync.Map{},
		EntryForwarder:     &KubectlForwarder{},
	}
	actual := NewEntryManager(out)
	testutil.CheckDeepEqual(t, expected, actual, cmp.AllowUnexported(EntryManager{}, sync.Map{}, sync.Mutex{}, atomic.Value{}))
}

func TestStop(t *testing.T) {
	pfe1 := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      constants.Pod,
			Name:      "resource",
			Namespace: "default",
		},
	}
	pfe2 := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      constants.Pod,
			Name:      "resource2",
			Namespace: "default",
		},
	}

	em := NewEntryManager(ioutil.Discard)

	em.forwardedResources = &sync.Map{}
	em.forwardedResources.Store("pod-resource-default-0", pfe1)
	em.forwardedResources.Store("pod-resource2-default-0", pfe2)

	fakeForwarder := newTestForwarder(nil)
	fakeForwarder.forwardedEntries = em.forwardedResources
	em.EntryForwarder = fakeForwarder

	em.Stop()

	if count := lengthSyncMap(fakeForwarder.forwardedEntries); count != 0 {
		t.Fatalf("error stopping port forwarding. expected 0 entries and got %d", count)
	}
}
