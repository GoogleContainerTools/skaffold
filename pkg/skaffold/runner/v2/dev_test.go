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

package v2

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	k8s "k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
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
	testBench *TestBench
}

func (t *TestMonitor) Register(_ func() ([]string, error), onChange func(filemon.Events)) error {
	t.callbacks = append(t.callbacks, onChange)
	return nil
}

func (t *TestMonitor) Run(bool) error {
	if t.testBench.intentTrigger {
		return nil
	}

	evt := t.events[t.testBench.currentCycle]

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

func mockK8sClient(string) (k8s.Interface, error) {
	return fakekubeclientset.NewSimpleClientset(), nil
}

func TestDevFailFirstCycle(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *TestBench
		monitor         filemon.Monitor
		expectedActions []Actions
	}{
		{
			description:     "fails to build the first time",
			testBench:       &TestBench{buildErrors: []error{errors.New("")}},
			monitor:         &NoopMonitor{},
			expectedActions: []Actions{{}},
		},
		{
			description: "fails to test the first time",
			testBench:   &TestBench{testErrors: []error{errors.New("")}},
			monitor:     &NoopMonitor{},
			expectedActions: []Actions{{
				Built: []string{"img:1"},
			}},
		},
		{
			description: "fails to render the first time",
			testBench:   &TestBench{renderErrors: []error{errors.New("")}},
			monitor:     &NoopMonitor{},
			expectedActions: []Actions{{
				Built:  []string{"img:1"},
				Tested: []string{"img:1"},
			}},
		},
		{
			description: "fails to deploy the first time",
			testBench:   &TestBench{deployErrors: []error{errors.New("")}},
			monitor:     &NoopMonitor{},
			expectedActions: []Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Rendered: []string{"img:1"},
			}},
		},
		{
			description: "fails to watch after first cycle",
			testBench:   &TestBench{},
			monitor:     &FailMonitor{},
			expectedActions: []Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Rendered: []string{"img:1"},
				Deployed: []string{"img:1"},
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)
			artifacts := []*latestV2.Artifact{{
				ImageName: "img",
			}}
			r := createRunner(t, test.testBench, test.monitor, artifacts, nil)
			test.testBench.firstMonitor = test.monitor.Run

			err := r.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckErrorAndDeepEqual(true, err, test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestDev(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *TestBench
		watchEvents     []filemon.Events
		expectedActions []Actions
	}{
		{
			description: "ignore subsequent build errors",
			testBench:   NewTestBench().WithBuildErrors([]error{nil, errors.New("")}),
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{},
			},
		},
		{
			description: "ignore subsequent test errors",
			testBench:   &TestBench{testErrors: []error{nil, errors.New("")}},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "ignore subsequent render errors",
			testBench:   &TestBench{renderErrors: []error{nil, errors.New("")}},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:  []string{"img1:2", "img2:2"},
					Tested: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "ignore subsequent deploy errors",
			testBench:   &TestBench{deployErrors: []error{nil, errors.New("")}},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:    []string{"img1:2", "img2:2"},
					Tested:   []string{"img1:2", "img2:2"},
					Rendered: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "full cycle twice",
			testBench:   &TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:    []string{"img1:2", "img2:2"},
					Tested:   []string{"img1:2", "img2:2"},
					Rendered: []string{"img1:2", "img2:2"},
					Deployed: []string{"img1:2", "img2:2"},
				},
			},
		},
		{
			description: "only change second artifact",
			testBench:   &TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:    []string{"img2:2"},
					Tested:   []string{"img2:2"},
					Rendered: []string{"img1:1", "img2:2"},
					Deployed: []string{"img1:1", "img2:2"},
				},
			},
		},
		{
			description: "redeploy",
			testBench:   &TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"manifest.yaml"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Rendered: []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)
			test.testBench.cycles = len(test.watchEvents)
			artifacts := []*latestV2.Artifact{
				{ImageName: "img1"},
				{ImageName: "img2"},
			}
			r := createRunner(t, test.testBench, &TestMonitor{
				events:    test.watchEvents,
				testBench: test.testBench,
			}, artifacts, nil)

			err := r.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestDevAutoTriggers(t *testing.T) {
	tests := []struct {
		description     string
		watchEvents     []filemon.Events
		expectedActions []Actions
		autoTriggers    triggerState // the state of auto triggers
		singleTriggers  triggerState // the state of single intent triggers at the end of dev loop
		userIntents     []func(i *runner.Intents)
	}{
		{
			description: "build on; sync on; deploy on",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:   triggerState{true, true, true},
			singleTriggers: triggerState{true, true, true},
			expectedActions: []Actions{
				{
					Synced: []string{"img1:1"},
				},
				{
					Built:    []string{"img2:2"},
					Tested:   []string{"img2:2"},
					Rendered: []string{"img1:1", "img2:2"},
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
			expectedActions: []Actions{{}, {}},
		},
		{
			description: "build on; sync off; deploy off",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:   triggerState{true, false, false},
			singleTriggers: triggerState{true, false, false},
			expectedActions: []Actions{{}, {
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
			autoTriggers:   triggerState{false, true, false},
			singleTriggers: triggerState{false, true, false},
			expectedActions: []Actions{{
				Synced: []string{"img1:1"},
			}, {}},
		},
		{
			description: "build off; sync off; deploy on",
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file2"}},
			},
			autoTriggers:    triggerState{false, false, true},
			singleTriggers:  triggerState{false, false, true},
			expectedActions: []Actions{{}, {}},
		},
		{
			description:     "build off; sync off; deploy off; user requests build, but no change so intent is discarded",
			watchEvents:     []filemon.Events{},
			autoTriggers:    triggerState{false, false, false},
			singleTriggers:  triggerState{false, false, false},
			expectedActions: []Actions{},
			userIntents: []func(i *runner.Intents){
				func(i *runner.Intents) {
					i.SetBuild(true)
				},
			},
		},
		{
			description:     "build off; sync off; deploy off; user requests build, and then sync, but no change so both intents are discarded",
			watchEvents:     []filemon.Events{},
			autoTriggers:    triggerState{false, false, false},
			singleTriggers:  triggerState{false, false, false},
			expectedActions: []Actions{},
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
			autoTriggers:    triggerState{false, false, false},
			singleTriggers:  triggerState{false, false, false},
			expectedActions: []Actions{},
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
	// first build-test-render-deploy sequence always happens
	firstAction := Actions{
		Built:    []string{"img1:1", "img2:1"},
		Tested:   []string{"img1:1", "img2:1"},
		Rendered: []string{"img1:1", "img2:1"},
		Deployed: []string{"img1:1", "img2:1"},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)
			t.Override(&sync.WorkingDir, func(context.Context, string, docker.Config) (string, error) { return "/", nil })
			testBench := &TestBench{}
			testBench.cycles = len(test.watchEvents)
			testBench.userIntents = test.userIntents
			artifacts := []*latestV2.Artifact{
				{
					ImageName: "img1",
					Sync: &latestV2.Sync{
						Manual: []*latestV2.SyncRule{{Src: "file1", Dest: "file1"}},
					},
				},
				{
					ImageName: "img2",
				},
			}
			r := createRunner(t, testBench, &TestMonitor{
				events:    test.watchEvents,
				testBench: testBench,
			}, artifacts, &test.autoTriggers)

			testBench.intents = r.intents

			err := r.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(append([]Actions{firstAction}, test.expectedActions...), testBench.Actions())

			build, _sync, deploy := r.intents.GetIntentsAttrs()
			singleTriggers := triggerState{
				build:  build,
				sync:   _sync,
				deploy: deploy,
			}
			t.CheckDeepEqual(test.singleTriggers, singleTriggers, cmp.AllowUnexported(triggerState{}))
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
		testBench                  *TestBench
		watchEvents                []filemon.Events
		expectedActions            []Actions
		expectedFileSyncEventCalls fileSyncEventCalls
	}{
		{
			description: "sync",
			testBench:   &TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
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
			testBench:   &TestBench{},
			watchEvents: []filemon.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file1"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Rendered: []string{"img1:1", "img2:1"},
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
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var actualFileSyncEventCalls fileSyncEventCalls
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)
			t.Override(&fileSyncInProgress, func(int, string) { actualFileSyncEventCalls.InProgress++ })
			t.Override(&fileSyncFailed, func(int, string, error) { actualFileSyncEventCalls.Failed++ })
			t.Override(&fileSyncSucceeded, func(int, string) { actualFileSyncEventCalls.Succeeded++ })
			t.Override(&sync.WorkingDir, func(context.Context, string, docker.Config) (string, error) { return "/", nil })
			test.testBench.cycles = len(test.watchEvents)
			artifacts := []*latestV2.Artifact{
				{
					ImageName: "img1",
					Sync: &latestV2.Sync{
						Manual: []*latestV2.SyncRule{{Src: "file1", Dest: "file1"}},
					},
				},
				{
					ImageName: "img2",
				},
			}
			r := createRunner(t, test.testBench, &TestMonitor{
				events:    test.watchEvents,
				testBench: test.testBench,
			}, artifacts, nil)

			err := r.Dev(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedActions, test.testBench.Actions())
			t.CheckDeepEqual(test.expectedFileSyncEventCalls, actualFileSyncEventCalls)
		})
	}
}
