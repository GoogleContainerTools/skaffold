/*
Copyright 2020 The Skaffold Authors

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

package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

// errMonitor is a filemon.Monitor that always fail to run.
type errMonitor struct {
	filemon.Monitor
}

func (f *errMonitor) Run(debounce bool) error {
	return errors.New("BUG")
}

// fakeMonitor is a filemon.Monitor that always succeeds.
type fakeMonitor struct {
	filemon.Monitor
}

func (f *fakeMonitor) Run(debounce bool) error {
	return nil
}

type fakeTrigger struct {
	trigger.Trigger
}

func (f *fakeTrigger) Debounce() bool {
	return false
}

func (f *fakeTrigger) LogWatchToUser(w io.Writer) {
	fmt.Fprintln(w, "Fake listener started")
}

func TestSkipDevLoopOnMonitorError(t *testing.T) {
	listener := &SkaffoldListener{
		Monitor: &errMonitor{},
		Trigger: &fakeTrigger{},
	}

	var devLoopWasCalled bool
	err := listener.do(func() error {
		devLoopWasCalled = true
		return nil
	})

	testutil.CheckErrorAndDeepEqual(t, false, err, false, devLoopWasCalled)
}

func TestContinueOnDevLoopError(t *testing.T) {
	listener := &SkaffoldListener{
		Monitor: &fakeMonitor{},
		Trigger: &fakeTrigger{},
	}

	err := listener.do(func() error {
		return errors.New("devloop error")
	})

	testutil.CheckError(t, false, err)
}

func TestReportDevLoopError(t *testing.T) {
	listener := &SkaffoldListener{
		Monitor: &fakeMonitor{},
		Trigger: &fakeTrigger{},
	}

	err := listener.do(func() error {
		return ErrorConfigurationChanged
	})

	if err != ErrorConfigurationChanged {
		t.Fatalf("should have returned a ErrorConfigurationChanged error, returned %v", err)
	}
}

func TestSkaffoldListener_LogWatch(t *testing.T) {
	out := new(bytes.Buffer)

	l := &SkaffoldListener{
		Trigger: &fakeTrigger{},
	}
	l.LogWatchIsActive(out)
	l.LogWatchIsInactive(out)
	got, want := out.String(), "Fake listener started\nNot watching for changes...\n"
	testutil.CheckDeepEqual(t, want, got)
}
