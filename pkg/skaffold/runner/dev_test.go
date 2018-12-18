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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type NoopWatcher struct{}

func (t *NoopWatcher) Register(deps func() ([]string, error), onChange func(watch.Events)) error {
	return nil
}

func (t *NoopWatcher) Run(ctx context.Context, trigger watch.Trigger, onChange func() error) error {
	return nil
}

type FailWatcher struct{}

func (t *FailWatcher) Register(deps func() ([]string, error), onChange func(watch.Events)) error {
	return nil
}

func (t *FailWatcher) Run(ctx context.Context, trigger watch.Trigger, onChange func() error) error {
	return errors.New("BUG")
}

type TestWatcher struct {
	events    []watch.Events
	callbacks []func(watch.Events)
}

func (t *TestWatcher) Register(deps func() ([]string, error), onChange func(watch.Events)) error {
	t.callbacks = append(t.callbacks, onChange)
	return nil
}

func (t *TestWatcher) Run(ctx context.Context, trigger watch.Trigger, onChange func() error) error {
	for _, evt := range t.events {
		for _, file := range evt.Modified {
			switch file {
			case "file1":
				t.callbacks[0](evt) // 1st artifact changed
			case "file2":
				t.callbacks[1](evt) // 2nd artifact changed
			case "manifest.yaml":
				t.callbacks[3](evt) // deployment configuration changed
			}
		}

		if err := onChange(); err != nil {
			return err
		}
	}

	return nil
}

func TestDevFailFirstCycle(t *testing.T) {
	var tests = []struct {
		description      string
		builder          *TestBuilder
		tester           *TestTester
		deployer         *TestDeployer
		watcher          watch.Watcher
		expectedBuilt    [][]string
		expectedTested   [][]string
		expectedDeployed [][]string
	}{
		{
			description: "fails to build the first time",
			builder:     &TestBuilder{errors: []error{errors.New("")}},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
			watcher:     &NoopWatcher{},
		},
		{
			description:   "fails to test the first time",
			builder:       &TestBuilder{},
			tester:        &TestTester{errors: []error{errors.New("")}},
			deployer:      &TestDeployer{},
			watcher:       &NoopWatcher{},
			expectedBuilt: [][]string{{"img:1"}},
		},
		{
			description:    "fails to deploy the first time",
			builder:        &TestBuilder{},
			tester:         &TestTester{},
			deployer:       &TestDeployer{errors: []error{errors.New("")}},
			watcher:        &NoopWatcher{},
			expectedBuilt:  [][]string{{"img:1"}},
			expectedTested: [][]string{{"img:1"}},
		},
		{
			description:      "fails to watch after first cycle",
			builder:          &TestBuilder{},
			tester:           &TestTester{},
			deployer:         &TestDeployer{},
			watcher:          &FailWatcher{},
			expectedBuilt:    [][]string{{"img:1"}},
			expectedTested:   [][]string{{"img:1"}},
			expectedDeployed: [][]string{{"img:1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createDefaultRunner(t)
			runner.Builder = test.builder
			runner.Tester = test.tester
			runner.Deployer = test.deployer
			runner.Watcher = test.watcher

			err := runner.Dev(context.Background(), ioutil.Discard, []*latest.Artifact{{
				ImageName: "img",
			}})

			testutil.CheckError(t, true, err)
			testutil.CheckDeepEqual(t, test.expectedBuilt, test.builder.built)
			testutil.CheckDeepEqual(t, test.expectedTested, test.tester.tested)
			testutil.CheckDeepEqual(t, test.expectedDeployed, test.deployer.deployed)
		})
	}
}

func TestDev(t *testing.T) {
	var tests = []struct {
		description      string
		builder          *TestBuilder
		tester           *TestTester
		deployer         *TestDeployer
		watchEvents      []watch.Events
		expectedBuilt    [][]string
		expectedTested   [][]string
		expectedDeployed [][]string
	}{
		{
			description: "ignore subsequent build errors",
			builder:     &TestBuilder{errors: []error{nil, errors.New("")}},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedBuilt:    [][]string{{"img1:1", "img2:1"}},
			expectedTested:   [][]string{{"img1:1", "img2:1"}},
			expectedDeployed: [][]string{{"img1:1", "img2:1"}},
		},
		{
			description: "ignore subsequent test errors",
			builder:     &TestBuilder{},
			tester:      &TestTester{errors: []error{nil, errors.New("")}},
			deployer:    &TestDeployer{},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedBuilt:    [][]string{{"img1:1", "img2:1"}, {"img1:2", "img2:2"}},
			expectedTested:   [][]string{{"img1:1", "img2:1"}},
			expectedDeployed: [][]string{{"img1:1", "img2:1"}},
		},
		{
			description: "ignore subsequent deploy errors",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{errors: []error{nil, errors.New("")}},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedBuilt:    [][]string{{"img1:1", "img2:1"}, {"img1:2", "img2:2"}},
			expectedTested:   [][]string{{"img1:1", "img2:1"}, {"img1:2", "img2:2"}},
			expectedDeployed: [][]string{{"img1:1", "img2:1"}},
		},
		{
			description: "full cycle twice",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedBuilt:    [][]string{{"img1:1", "img2:1"}, {"img1:2", "img2:2"}},
			expectedTested:   [][]string{{"img1:1", "img2:1"}, {"img1:2", "img2:2"}},
			expectedDeployed: [][]string{{"img1:1", "img2:1"}, {"img1:2", "img2:2"}},
		},
		{
			description: "only change second artifact",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
			watchEvents: []watch.Events{
				{Modified: []string{"file2"}},
			},
			expectedBuilt:    [][]string{{"img1:1", "img2:1"}, {"img2:2"}},
			expectedTested:   [][]string{{"img1:1", "img2:1"}, {"img2:2"}},
			expectedDeployed: [][]string{{"img1:1", "img2:1"}, {"img2:2", "img1:1"}},
		},
		{
			description: "redeploy",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
			watchEvents: []watch.Events{
				{Modified: []string{"manifest.yaml"}},
			},
			expectedBuilt:    [][]string{{"img1:1", "img2:1"}},
			expectedTested:   [][]string{{"img1:1", "img2:1"}},
			expectedDeployed: [][]string{{"img1:1", "img2:1"}, {"img1:1", "img2:1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createDefaultRunner(t)
			runner.Builder = test.builder
			runner.Tester = test.tester
			runner.Deployer = test.deployer
			runner.Watcher = &TestWatcher{
				events: test.watchEvents,
			}

			err := runner.Dev(context.Background(), ioutil.Discard, []*latest.Artifact{
				{ImageName: "img1"},
				{ImageName: "img2"},
			})

			testutil.CheckError(t, false, err)
			testutil.CheckDeepEqual(t, test.expectedBuilt, test.builder.built)
			testutil.CheckDeepEqual(t, test.expectedTested, test.tester.tested)
			testutil.CheckDeepEqual(t, test.expectedDeployed, test.deployer.deployed)
		})
	}
}
