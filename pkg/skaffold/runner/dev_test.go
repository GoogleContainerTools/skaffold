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

package runner

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/client-go/tools/clientcmd/api"
)

type NoopWatcher struct{}

func (t *NoopWatcher) Register(func() ([]string, error), func(watch.Events)) error {
	return nil
}

func (t *NoopWatcher) Run(context.Context, io.Writer, func() error) error {
	return nil
}

type FailWatcher struct{}

func (t *FailWatcher) Register(func() ([]string, error), func(watch.Events)) error {
	return nil
}

func (t *FailWatcher) Run(context.Context, io.Writer, func() error) error {
	return errors.New("BUG")
}

type TestWatcher struct {
	events    []watch.Events
	callbacks []func(watch.Events)
	testBench *TestBench
}

func (t *TestWatcher) Register(deps func() ([]string, error), onChange func(watch.Events)) error {
	t.callbacks = append(t.callbacks, onChange)
	return nil
}

func (t *TestWatcher) Run(ctx context.Context, out io.Writer, onChange func() error) error {
	for _, evt := range t.events {
		t.testBench.enterNewCycle()

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
		description     string
		testBench       *TestBench
		watcher         watch.Watcher
		expectedActions []Actions
	}{
		{
			description:     "fails to build the first time",
			testBench:       &TestBench{buildErrors: []error{errors.New("")}},
			watcher:         &NoopWatcher{},
			expectedActions: []Actions{{}},
		},
		{
			description: "fails to test the first time",
			testBench:   &TestBench{testErrors: []error{errors.New("")}},
			watcher:     &NoopWatcher{},
			expectedActions: []Actions{{
				Built: []string{"img:1"},
			}},
		},
		{
			description: "fails to deploy the first time",
			testBench:   &TestBench{deployErrors: []error{errors.New("")}},
			watcher:     &NoopWatcher{},
			expectedActions: []Actions{{
				Built:  []string{"img:1"},
				Tested: []string{"img:1"},
			}},
		},
		{
			description: "fails to watch after first cycle",
			testBench:   &TestBench{},
			watcher:     &FailWatcher{},
			expectedActions: []Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Deployed: []string{"img:1"},
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			runner := createRunner(t, test.testBench)
			runner.Watcher = test.watcher

			err := runner.Dev(context.Background(), ioutil.Discard, []*latest.Artifact{{
				ImageName: "img",
			}})

			t.CheckErrorAndDeepEqual(true, err, test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestDev(t *testing.T) {
	var tests = []struct {
		description     string
		testBench       *TestBench
		watchEvents     []watch.Events
		expectedActions []Actions
	}{
		{
			description: "ignore subsequent build errors",
			testBench:   &TestBench{buildErrors: []error{nil, errors.New("")}},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
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
			testBench:   &TestBench{testErrors: []error{nil, errors.New("")}},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
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
			testBench:   &TestBench{deployErrors: []error{nil, errors.New("")}},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
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
			testBench:   &TestBench{},
			watchEvents: []watch.Events{
				{Modified: []string{"file1", "file2"}},
			},
			expectedActions: []Actions{
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
			testBench:   &TestBench{},
			watchEvents: []watch.Events{
				{Modified: []string{"file2"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Built:    []string{"img2:2"},
					Tested:   []string{"img2:2"},
					Deployed: []string{"img2:2", "img1:1"},
				},
			},
		},
		{
			description: "redeploy",
			testBench:   &TestBench{},
			watchEvents: []watch.Events{
				{Modified: []string{"manifest.yaml"}},
			},
			expectedActions: []Actions{
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
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			runner := createRunner(t, test.testBench)
			runner.Watcher = &TestWatcher{
				events:    test.watchEvents,
				testBench: test.testBench,
			}

			err := runner.Dev(context.Background(), ioutil.Discard, []*latest.Artifact{
				{ImageName: "img1"},
				{ImageName: "img2"},
			})

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestDevSync(t *testing.T) {
	var tests = []struct {
		description     string
		testBench       *TestBench
		watchEvents     []watch.Events
		expectedActions []Actions
	}{
		{
			description: "sync",
			testBench:   &TestBench{},
			watchEvents: []watch.Events{
				{Modified: []string{"file1"}},
			},
			expectedActions: []Actions{
				{
					Built:    []string{"img1:1", "img2:1"},
					Tested:   []string{"img1:1", "img2:1"},
					Deployed: []string{"img1:1", "img2:1"},
				},
				{
					Synced: []string{"img1:1"},
				},
			},
		},
		{
			description: "sync twice",
			testBench:   &TestBench{},
			watchEvents: []watch.Events{
				{Modified: []string{"file1"}},
				{Modified: []string{"file1"}},
			},
			expectedActions: []Actions{
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
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&sync.WorkingDir, func(string, map[string]bool) (string, error) { return "/", nil })

			runner := createRunner(t, test.testBench)
			runner.Watcher = &TestWatcher{
				events:    test.watchEvents,
				testBench: test.testBench,
			}

			err := runner.Dev(context.Background(), ioutil.Discard, []*latest.Artifact{
				{
					ImageName: "img1",
					Sync: &latest.Sync{
						Manual: []*latest.SyncRule{{Src: "file1", Dest: "file1"}},
					},
				},
				{
					ImageName: "img2",
				},
			})

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedActions, test.testBench.Actions())
		})
	}
}
