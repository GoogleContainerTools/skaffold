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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestStop(t *testing.T) {
	event.InitializeState(latest.Pipeline{}, "test", true, true, true)

	pfe1 := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      constants.Pod,
		Name:      "resource",
		Namespace: "default",
	}, "", "", "", "", 9000, false)

	pfe2 := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      constants.Pod,
		Name:      "resource2",
		Namespace: "default",
	}, "", "", "", "", 9001, false)

	fakeForwarder := newTestForwarder()
	em := NewEntryManager(ioutil.Discard, fakeForwarder)
	em.forwardPortForwardEntry(context.Background(), pfe1)
	em.forwardPortForwardEntry(context.Background(), pfe2)

	testutil.CheckDeepEqual(t, 2, fakeForwarder.forwardedResources.Length())
	testutil.CheckDeepEqual(t, 2, fakeForwarder.forwardedPorts.Length())

	em.Stop()

	testutil.CheckDeepEqual(t, 0, fakeForwarder.forwardedResources.Length())
	testutil.CheckDeepEqual(t, 0, fakeForwarder.forwardedPorts.Length())
}

func TestForwardedResources(t *testing.T) {
	pf := &forwardedResources{}

	// Try to store a resource
	pf.Store("resource", &portForwardEntry{})

	// Try to load the resource
	if _, ok := pf.Load("resource"); !ok {
		t.Fatal("didn't load resource correctly correctly")
	}

	// Try to load a resource that doesn't exist
	if actual, ok := pf.Load("dne"); ok || actual != nil {
		t.Fatal("loaded resource that doesn't exist")
	}
}
