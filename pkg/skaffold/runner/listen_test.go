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
	"context"
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/trigger"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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

type fakeTriggger struct {
	trigger.Trigger
}

func (f *fakeTriggger) Debounce() bool {
	return false
}

type fakeDepsResolver struct{}

func (f *fakeDepsResolver) TransitiveArtifactDependencies(context.Context, *latest.Artifact) ([]string, error) {
	return nil, nil
}

func (f *fakeDepsResolver) SingleArtifactDependencies(context.Context, *latest.Artifact) ([]string, error) {
	return nil, nil
}

func (f *fakeDepsResolver) Reset() {}

func TestSkipDevLoopOnMonitorError(t *testing.T) {
	listener := &SkaffoldListener{
		Monitor:                 &errMonitor{},
		Trigger:                 &fakeTriggger{},
		sourceDependenciesCache: &fakeDepsResolver{},
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
		Monitor:                 &fakeMonitor{},
		Trigger:                 &fakeTriggger{},
		sourceDependenciesCache: &fakeDepsResolver{},
	}

	err := listener.do(func() error {
		return errors.New("devloop error")
	})

	testutil.CheckError(t, false, err)
}

func TestReportDevLoopError(t *testing.T) {
	listener := &SkaffoldListener{
		Monitor:                 &fakeMonitor{},
		Trigger:                 &fakeTriggger{},
		sourceDependenciesCache: &fakeDepsResolver{},
	}

	err := listener.do(func() error {
		return ErrorConfigurationChanged
	})

	if err != ErrorConfigurationChanged {
		t.Fatalf("should have returned a ErrorConfigurationChanged error, returned %v", err)
	}
}
