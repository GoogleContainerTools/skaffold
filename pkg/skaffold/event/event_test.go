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
)

func TestGetLogEvents(t *testing.T) {
	for step := 0; step < 100000; step++ {
		ev := &eventHandler{}

		ev.logEvent(proto.LogEntry{Entry: "OLD1"})

		var done int32
		go func() {
			ev.logEvent(proto.LogEntry{Entry: "FRESH"})

			for atomic.LoadInt32(&done) == 0 {
				ev.logEvent(proto.LogEntry{Entry: "POISON PILL"})
			}
		}()

		received := 0
		ev.forEachEvent(func(e *proto.LogEntry) error {
			if e.Entry == "POISON PILL" {
				return errors.New("Done")
			}

			received++
			return nil
		})
		atomic.StoreInt32(&done, int32(1))

		if received != 2 {
			t.Fatalf("Expected %d events, Got %d (Step: %d)", 2, received, step)
		}
	}
}
