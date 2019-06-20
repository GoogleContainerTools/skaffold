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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server/proto"
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

func TestDeployInProgress(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(nil),
	}

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployInProgress()
	wait(t, func() bool { return handler.getState().DeployState.Status == InProgress })
}

func TestDeployFailed(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(nil),
	}

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployFailed(errors.New("BUG"))
	wait(t, func() bool { return handler.getState().DeployState.Status == Failed })
}

func TestDeployComplete(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(nil),
	}

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployComplete()
	wait(t, func() bool { return handler.getState().DeployState.Status == Complete })
}

func TestBuildInProgress(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(&latest.BuildConfig{
			Artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
		}),
	}

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildInProgress("img")
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == InProgress })
}

func TestBuildFailed(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(&latest.BuildConfig{
			Artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
		}),
	}

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildFailed("img", errors.New("BUG"))
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == Failed })
}

func TestBuildComplete(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(&latest.BuildConfig{
			Artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
		}),
	}

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildComplete("img")
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == Complete })
}

func TestPortForwarded(t *testing.T) {
	defer func() { handler = nil }()

	handler = &eventHandler{
		state: emptyState(nil),
	}

	wait(t, func() bool { return handler.getState().ForwardedPorts["container"] == nil })
	PortForwarded(8080, 8888, "pod", "container", "ns", "portname", "resourceType", "resourceName")
	wait(t, func() bool { return handler.getState().ForwardedPorts["container"] != nil })
}

func wait(t *testing.T, condition func() bool) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}

		case <-timeout.C:
			t.Fatal("Timed out waiting")
		}
	}
}
