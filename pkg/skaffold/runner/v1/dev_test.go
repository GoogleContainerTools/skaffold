/*
Copyright 2021 The Skaffold Authors

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

package v1

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/test"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type NoopMonitor struct{}

func (t *NoopMonitor) Register(func() ([]string, error), func(filemon.Events)) error {
	return nil
}

func (t *NoopMonitor) Run(bool) error {
	return nil
}

func (t *NoopMonitor) Reset() {}

type FailMonitor struct{}

func (t *FailMonitor) Register(func() ([]string, error), func(filemon.Events)) error {
	return nil
}

func (t *FailMonitor) Run(bool) error {
	return errors.New("BUG")
}

func (t *FailMonitor) Reset() {}

type TestMonitor struct {
	events    []filemon.Events
	callbacks []func(filemon.Events)
	testBench *test.TestBench
}

func (t *TestMonitor) Register(deps func() ([]string, error), onChange func(filemon.Events)) error {
	t.callbacks = append(t.callbacks, onChange)
	return nil
}

func (t *TestMonitor) Run(bool) error {
	if t.testBench.IntentTrigger {
		return nil
	}

	evt := t.events[t.testBench.CurrentCycle]

	for _, file := range evt.Modified {
		switch file {
		case "file1":
			t.callbacks[0](evt) // 1st artifact changed
		case "file2":
			t.callbacks[1](evt) // 2nd artifact changed
		// callbacks[2] and callbacks[3] are for `test` dependency triggers
		case "manifest.yaml":
			t.callbacks[4](evt) // deployment configuration changed
		}
	}

	return nil
}

func (t *TestMonitor) Reset() {}

func TestDevFailFirstCycle(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *test.TestBench
		monitor         filemon.Monitor
		expectedActions []test.Actions
	}{
		{
			description:     "fails to build the first time",
			testBench:       &test.TestBench{BuildErrors: []error{errors.New("")}},
			monitor:         &NoopMonitor{},
			expectedActions: []test.Actions{{}},
		},
		{
			description: "fails to test the first time",
			testBench:   &test.TestBench{TestErrors: []error{errors.New("")}},
			monitor:     &NoopMonitor{},
			expectedActions: []test.Actions{{
				Built: []string{"img:1"},
			}},
		},
		{
			description: "fails to deploy the first time",
			testBench:   &test.TestBench{DeployErrors: []error{errors.New("")}},
			monitor:     &NoopMonitor{},
			expectedActions: []test.Actions{{
				Built:  []string{"img:1"},
				Tested: []string{"img:1"},
			}},
		},
		{
			description: "fails to watch after first cycle",
			testBench:   &test.TestBench{},
			monitor:     &FailMonitor{},
			expectedActions: []test.Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Deployed: []string{"img:1"},
			}},
		},
	}
	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)
			artifacts := []*latest_v1.Artifact{{
				ImageName: "img",
			}}
			r := MockRunnerV1(t, testdata.testBench, testdata.monitor, artifacts, nil)
			testdata.testBench.FirstMonitor = testdata.monitor.Run

			err := r.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckErrorAndDeepEqual(true, err, testdata.expectedActions, testdata.testBench.Actions())
		})
	}
}

func TestDev(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *test.TestBench
		watchEvents     []filemon.Events
		expectedActions []test.Actions
	}{
		{
			description: "ignore subsequent build errors",
			testBench:   test.NewTestBench().WithBuildErrors([]error{nil, errors.New("")}),
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{},
			},
		},
		{
			description: "ignore subsequent test errors",
			testBench:   &test.TestBench{TestErrors: []error{nil, errors.New("")}},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "ignore subsequent deploy errors",
			testBench:   &test.TestBench{DeployErrors: []error{nil, errors.New("")}},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:  []string{"img1:2", "img2:2"},
					Tested: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "full cycle twice",
			testBench:   &test.TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:    []string{"img1:2", "img2:2"},
					Tested:   []string{"img1:2", "img2:2"},
					Deployed: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "only change second artifact",
			testBench:   &test.TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file2"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:    []string{"img2:2"},
					Tested:   []string{"img2:2"},
					Deployed: []string{"img1:1", "img2:2"},
				},
			},
		},
		{
			description: "redeploy",
			testBench:   &test.TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"manifest.yaml"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Deployed: []string{"img1:1", "img2:1"},
				},
			},
		},
	}
	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)
			testdata.testBench.Cycles = len(testdata.watchEvents)
			artifacts := []*latest_v1.Artifact{
				{ImageName: "img1"},
				{ImageName: "img2"},
			}
			r := MockRunnerV1(t, testdata.testBench, &TestMonitor{
				events:    testdata.watchEvents,
				testBench: testdata.testBench,
			}, artifacts, nil)

			err := r.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(testdata.expectedActions, testdata.testBench.Actions())
		})
	}
}

func TestDevAutoTriggers(t *testing.T) {
	tests := []struct {
		description     string
		watchEvents     []filemon.Events
		expectedActions []test.Actions
		autoTriggers    test.TriggerState // the state of auto triggers
		singleTriggers  test.TriggerState // the state of single intent triggers at the end of dev loop
		userIntents     []func(i *runner.Intents)
	}{
		{
			description: "build on; sync on; deploy on",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:   test.TriggerState{true, true, true},
			singleTriggers: test.TriggerState{true, true, true},
			expectedActions: []test.Actions{
				{
					Synced: []string{"img1:1"},
				},
				{
					Built:    []string{"img2:2"},
					Tested:   []string{"img2:2"},
					Deployed: []string{"img1:1", "img2:2"},
				},
			},
		},
		{
			description: "build off; sync off; deploy off",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			expectedActions: []test.Actions{{}, {}},
		},
		{
			description: "build on; sync off; deploy off",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:   test.TriggerState{true, false, false},
			singleTriggers: test.TriggerState{true, false, false},
			expectedActions: []test.Actions{{}, {
				Built:  []string{"img2:2"},
				Tested: []string{"img2:2"},
			}},
		},
		{
			description: "build off; sync on; deploy off",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:   test.TriggerState{false, true, false},
			singleTriggers: test.TriggerState{false, true, false},
			expectedActions: []test.Actions{{
				Synced: []string{"img1:1"},
			}, {}},
		},
		{
			description: "build off; sync off; deploy on",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:    test.TriggerState{false, false, true},
			singleTriggers:  test.TriggerState{false, false, true},
			expectedActions: []test.Actions{{}, {}},
		},
		{
			description:     "build off; sync off; deploy off; user requests build, but no change so intent is discarded",
			watchEvents:     []filemon.Events{},
			autoTriggers:    test.TriggerState{false, false, false},
			singleTriggers:  test.TriggerState{false, false, false},
			expectedActions: []test.Actions{},
			userIntents: []func(i *runner.Intents){
				func(i *runner.Intents) {
					i.SetBuild(true)
				},
			},
		},
		{
			description:     "build off; sync off; deploy off; user requests build, and then sync, but no change so both intents are discarded",
			watchEvents:     []filemon.Events{},
			autoTriggers:    test.TriggerState{false, false, false},
			singleTriggers:  test.TriggerState{false, false, false},
			expectedActions: []test.Actions{},
			userIntents: []func(i *runner.Intents){
				func(i *runner.Intents) {
					i.SetBuild(true)
					i.SetSync(true)
				},
			},
		},
		{
			description:     "build off; sync off; deploy off; user requests build, and then sync, but no change so both intents are discarded",
			watchEvents:     []filemon.Events{},
			autoTriggers:    test.TriggerState{false, false, false},
			singleTriggers:  test.TriggerState{false, false, false},
			expectedActions: []test.Actions{},
			userIntents: []func(i *runner.Intents){
				func(i *runner.Intents) {
					i.SetBuild(true)
				},
				func(i *runner.Intents) {
					i.SetSync(true)
				},
			},
		},
	}
	// first build-test-deploy sequence always happens
	firstAction := test.Actions{
		Built:    []string{"img1:1", "img2:1"},
		Tested:   []string{"img1:1", "img2:1"},
		Deployed: []string{"img1:1", "img2:1"},
	}

	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)
			t.Override(&sync.WorkingDir, func(string, docker.Config) (string, error) { return "/", nil })
			testBench := &test.TestBench{}
			testBench.Cycles = len(testdata.watchEvents)
			testBench.UserIntents = testdata.userIntents
			artifacts := []*latest_v1.Artifact{
				{
					ImageName: "img1",
					Sync: &latest_v1.Sync{
						Manual: []*latest_v1.SyncRule{{Src: "file1", Dest: "file1"}},
					},
				},
				{
					ImageName: "img2",
				},
			}
			mockedRunner := MockRunnerV1(t, testBench, &TestMonitor{
				events:    testdata.watchEvents,
				testBench: testBench,
			}, artifacts, &testdata.autoTriggers)

			testBench.Intents = mockedRunner.intents

			err := mockedRunner.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(append([]test.Actions{firstAction}, testdata.expectedActions...), testBench.Actions())

			build, sync, deploy := runner.GetIntentsAttrs(*mockedRunner.intents)
			singleTriggers := test.TriggerState{
				Build:  build,
				Sync:   sync,
				Deploy: deploy,
			}
			t.CheckDeepEqual(testdata.singleTriggers, singleTriggers, cmp.AllowUnexported(test.TriggerState{}))
		})
	}
}

func TestDevSync(t *testing.T) {
	type fileSyncEventCalls struct {
		InProgress int
		Failed     int
		Succeeded  int
	}

	tests := []struct {
		description                string
		testBench                  *test.TestBench
		watchEvents                []filemon.Events
		expectedActions            []test.Actions
		expectedFileSyncEventCalls fileSyncEventCalls
	}{
		{
			description: "sync",
			testBench:   &test.TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Synced: []string{"img1:1"},
				},
			},
			expectedFileSyncEventCalls: fileSyncEventCalls{
				InProgress: 1,
				Failed:     0,
				Succeeded:  1,
			},
		},
		{
			description: "sync twice",
			testBench:   &test.TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file1"}},
			},
			expectedActions: []test.Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Synced: []string{"img1:1"},
				},
				{
					Synced: []string{"img1:1"},
				},
			},
			expectedFileSyncEventCalls: fileSyncEventCalls{
				InProgress: 2,
				Failed:     0,
				Succeeded:  2,
			},
		},
	}
	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			var actualFileSyncEventCalls fileSyncEventCalls
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)
			t.Override(&fileSyncInProgress, func(int, string) { actualFileSyncEventCalls.InProgress++ })
			t.Override(&fileSyncFailed, func(int, string, error) { actualFileSyncEventCalls.Failed++ })
			t.Override(&fileSyncSucceeded, func(int, string) { actualFileSyncEventCalls.Succeeded++ })
			t.Override(&sync.WorkingDir, func(string, docker.Config) (string, error) { return "/", nil })
			testdata.testBench.Cycles = len(testdata.watchEvents)
			artifacts := []*latest_v1.Artifact{
				{
					ImageName: "img1",
					Sync: &latest_v1.Sync{
						Manual: []*latest_v1.SyncRule{{Src: "file1", Dest: "file1"}},
					},
				},
				{
					ImageName: "img2",
				},
			}
			runner := MockRunnerV1(t, testdata.testBench, &TestMonitor{
				events:    testdata.watchEvents,
				testBench: testdata.testBench,
			}, artifacts, nil)

			err := runner.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(testdata.expectedActions, testdata.testBench.Actions())
			t.CheckDeepEqual(testdata.expectedFileSyncEventCalls, actualFileSyncEventCalls)
		})
	}
}
