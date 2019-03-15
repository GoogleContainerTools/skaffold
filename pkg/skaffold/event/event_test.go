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

package event

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetLogEvents(t *testing.T) {
	for step := 0; step < 10000; step++ {
		ev := &eventHandler{}

		ev.logEvent(proto.LogEntry{Entry: "OLD"})
		go func() {
			ev.logEvent(proto.LogEntry{Entry: "FRESH"})
			ev.logEvent(proto.LogEntry{Entry: "POISON PILL"})
		}()

		var received int32
		ev.forEachEvent(func(e *proto.LogEntry) error {
			if e.Entry == "POISON PILL" {
				return errors.New("Done")
			}

			atomic.AddInt32(&received, 1)
			return nil
		})

		if atomic.LoadInt32(&received) != 2 {
			t.Fatalf("Expected %d events, Got %d (Step: %d)", 2, received, step)
		}
	}
}

func TestGetState(t *testing.T) {
	ev := &eventHandler{
		state: emptyState(nil),
	}

	ev.stateLock.Lock()
	ev.state.BuildState.Artifacts["img"] = Complete
	ev.stateLock.Unlock()

	state := ev.getState()

	testutil.CheckDeepEqual(t, Complete, state.BuildState.Artifacts["img"])
}
