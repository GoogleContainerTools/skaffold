/*
Copyright 2018 The Skaffold Authors

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
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type TestWatcher struct {
	changedArtifacts [][]int
	changeCallbacks  []func(watch.Events)
	events           []watch.Events
	err              error
}

func NewWatcherFactory(err error, events []watch.Events, changedArtifacts ...[]int) watch.Factory {
	return func() watch.Watcher {
		return &TestWatcher{
			changedArtifacts: changedArtifacts,
			events:           events,
			err:              err,
		}
	}
}

func (t *TestWatcher) Register(deps func() ([]string, error), onChange func(watch.Events)) error {
	t.changeCallbacks = append(t.changeCallbacks, onChange)
	return nil
}

func (t *TestWatcher) Run(ctx context.Context, trigger watch.Trigger, onChange func() error) error {
	evts := watch.Events{}
	if t.events != nil {
		evts = t.events[0]
		t.events = t.events[1:]
	}

	for _, artifactIndices := range t.changedArtifacts {
		for _, artifactIndex := range artifactIndices {
			t.changeCallbacks[artifactIndex](evts)
		}
		onChange()
	}
	return t.err
}

func TestDev(t *testing.T) {
	var tests = []struct {
		description    string
		builder        build.Builder
		tester         test.Tester
		deployer       deploy.Deployer
		watcherFactory watch.Factory
		shouldErr      bool
	}{
		{
			description: "fails to build the first time",
			builder: &TestBuilder{
				errors: []error{errors.New("")},
			},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "fails to deploy the first time",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer: &TestDeployer{
				errors: []error{errors.New("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "fails to test the first time",
			builder:     &TestBuilder{},
			tester: &TestTester{
				errors: []error{errors.New("")},
			},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil),
			shouldErr:      true,
		},
		{
			description: "ignore subsequent build errors",
			builder: &TestBuilder{
				errors: []error{nil, errors.New("")},
			},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(nil, nil, nil),
		},
		{
			description: "ignore subsequent deploy errors",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer: &TestDeployer{
				errors: []error{nil, errors.New("")},
			},
			watcherFactory: NewWatcherFactory(nil, nil, nil),
		},
		{
			description:    "fail to watch files",
			builder:        &TestBuilder{},
			tester:         &TestTester{},
			deployer:       &TestDeployer{},
			watcherFactory: NewWatcherFactory(errors.New(""), nil),
			shouldErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createDefaultRunner(t)
			runner.Builder = test.builder
			runner.Tester = test.tester
			runner.Deployer = test.deployer
			runner.watchFactory = test.watcherFactory

			err := runner.Dev(context.Background(), ioutil.Discard, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestBuildAndDeployAllArtifacts(t *testing.T) {
	builder := &TestBuilder{}
	deployer := &TestDeployer{}
	artifacts := []*latest.Artifact{
		{ImageName: "image1"},
		{ImageName: "image2"},
	}

	runner := createDefaultRunner(t)
	runner.Builder = builder
	runner.Deployer = deployer

	ctx := context.Background()

	// Both artifacts are changed
	runner.watchFactory = NewWatcherFactory(nil, nil, []int{0, 1})
	err := runner.Dev(ctx, ioutil.Discard, artifacts)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(builder.built) != 2 {
		t.Errorf("Expected 2 artifacts to be built. Got %d", len(builder.built))
	}
	if len(deployer.deployed) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed))
	}

	// Only one is changed
	runner.watchFactory = NewWatcherFactory(nil, nil, []int{1})
	err = runner.Dev(ctx, ioutil.Discard, artifacts)

	if err != nil {
		t.Errorf("Didn't expect an error. Got %s", err)
	}
	if len(builder.built) != 1 {
		t.Errorf("Expected 1 artifact to be built. Got %d", len(builder.built))
	}
	if len(deployer.deployed) != 2 {
		t.Errorf("Expected 2 artifacts to be deployed. Got %d", len(deployer.deployed))
	}
}
