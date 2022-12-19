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
	"strings"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

func TestStop(t *testing.T) {
	testEvent.InitializeState([]latest.Pipeline{{}})

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
	em := NewEntryManager(fakeForwarder)
	em.forwardPortForwardEntry(context.Background(), io.Discard, pfe1)
	em.forwardPortForwardEntry(context.Background(), io.Discard, pfe2)

	testutil.CheckDeepEqual(t, 2, length(&fakeForwarder.forwardedResources))
	testutil.CheckDeepEqual(t, 2, fakeForwarder.forwardedPorts.Length())

	em.Stop()

	testutil.CheckDeepEqual(t, 0, length(&fakeForwarder.forwardedResources))
	testutil.CheckDeepEqual(t, 0, fakeForwarder.forwardedPorts.Length())
}

// length returns the number of elements in a sync.Map
func length(m *sync.Map) int {
	n := 0
	m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	return n
}

// print is a String() function for a sync.Map
func print(m *sync.Map) string {
	var b strings.Builder
	b.WriteString("map[")
	n := 0
	m.Range(func(k, v interface{}) bool {
		if n > 0 {
			b.WriteRune(' ')
		}
		b.WriteString(fmt.Sprintf("%v:%v", k, v))
		n++
		return true
	})
	b.WriteRune(']')
	return b.String()
}
