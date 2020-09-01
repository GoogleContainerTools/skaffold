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

package trigger

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/rjeczalik/notify"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewTrigger(t *testing.T) {
	tests := []struct {
		description string
		opts        config.SkaffoldOptions
		expected    Trigger
		shouldErr   bool
	}{
		{
			description: "polling trigger",
			opts:        config.SkaffoldOptions{Trigger: "polling", WatchPollInterval: 1},
			expected: &pollTrigger{
				Interval: 1 * time.Millisecond,
			},
		},
		{
			description: "notify trigger",
			opts:        config.SkaffoldOptions{Trigger: "notify", WatchPollInterval: 1},
			expected: &fsNotifyTrigger{
				Interval: 1 * time.Millisecond,
				workspaces: map[string]struct{}{
					"../workspace":            {},
					"../some/other/workspace": {},
				},
				watchFunc: notify.Watch,
			},
		},
		{
			description: "manual trigger",
			opts:        config.SkaffoldOptions{Trigger: "manual"},
			expected:    &manualTrigger{},
		},
		{
			description: "unknown trigger",
			opts:        config.SkaffoldOptions{Trigger: "unknown"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runCtx := &runcontext.RunContext{
				Opts: test.opts,
				Cfg: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								Workspace: "../workspace",
							}, {
								Workspace: "../workspace",
							}, {
								Workspace: "../some/other/workspace",
							},
						},
					},
				},
			}

			got, err := NewTrigger(runCtx, nil)
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expected, got, cmp.AllowUnexported(fsNotifyTrigger{}), cmp.Comparer(ignoreFuncComparer), cmp.AllowUnexported(manualTrigger{}), cmp.AllowUnexported(pollTrigger{}))
			}
		})
	}
}

func ignoreFuncComparer(x, y func(path string, c chan<- notify.EventInfo, events ...notify.Event) error) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	return true // cannot assert function equality, so skip
}

func TestPollTrigger_Debounce(t *testing.T) {
	trigger := &pollTrigger{}
	got, want := trigger.Debounce(), true
	testutil.CheckDeepEqual(t, want, got)
}

func TestPollTrigger_LogWatchToUser(t *testing.T) {
	tests := []struct {
		description string
		isActive    bool
		expected    string
	}{
		{
			description: "active polling trigger",
			isActive:    true,
			expected:    "Watching for changes every 10ns...\n",
		},
		{
			description: "inactive polling trigger",
			isActive:    false,
			expected:    "Not watching for changes...\n",
		},
	}
	for _, test := range tests {
		out := new(bytes.Buffer)

		trigger := &pollTrigger{
			Interval: 10,
			isActive: func() bool {
				return test.isActive
			},
		}
		trigger.LogWatchToUser(out)

		got, want := out.String(), test.expected
		testutil.CheckDeepEqual(t, want, got)
	}
}

func TestNotifyTrigger_Debounce(t *testing.T) {
	trigger := &fsNotifyTrigger{}
	got, want := trigger.Debounce(), false
	testutil.CheckDeepEqual(t, want, got)
}

func TestNotifyTrigger_LogWatchToUser(t *testing.T) {
	tests := []struct {
		description string
		isActive    bool
		expected    string
	}{
		{
			description: "active notify trigger",
			isActive:    true,
			expected:    "Watching for changes...\n",
		},
		{
			description: "inactive notify trigger",
			isActive:    false,
			expected:    "Not watching for changes...\n",
		},
	}
	for _, test := range tests {
		out := new(bytes.Buffer)

		trigger := &fsNotifyTrigger{
			Interval: 10,
			isActive: func() bool {
				return test.isActive
			},
		}
		trigger.LogWatchToUser(out)

		got, want := out.String(), test.expected
		testutil.CheckDeepEqual(t, want, got)
	}
}

func TestManualTrigger_Debounce(t *testing.T) {
	trigger := &manualTrigger{}
	got, want := trigger.Debounce(), false
	testutil.CheckDeepEqual(t, want, got)
}

func TestManualTrigger_LogWatchToUser(t *testing.T) {
	tests := []struct {
		description string
		isActive    bool
		expected    string
	}{
		{
			description: "active manual trigger",
			isActive:    true,
			expected:    "Press any key to rebuild/redeploy the changes\n",
		},
		{
			description: "inactive manual trigger",
			isActive:    false,
			expected:    "Not watching for changes...\n",
		},
	}
	for _, test := range tests {
		out := new(bytes.Buffer)

		trigger := &manualTrigger{
			isActive: func() bool {
				return test.isActive
			},
		}
		trigger.LogWatchToUser(out)

		got, want := out.String(), test.expected
		testutil.CheckDeepEqual(t, want, got)
	}
}

func TestStartTrigger(t *testing.T) {
	tests := []struct {
		description string
		trigger     Trigger
	}{
		{
			description: "fsNotify trigger works",
			trigger: &fsNotifyTrigger{
				Interval:   2 * time.Second,
				workspaces: nil,
				isActive:   func() bool { return false },
				watchFunc: func(string, chan<- notify.EventInfo, ...notify.Event) error {
					return nil
				},
			},
		},
		{
			description: "fallback on polling trigger",
			trigger: &fsNotifyTrigger{
				Interval:   200 * time.Millisecond,
				workspaces: nil,
				isActive:   func() bool { return false },
				watchFunc: func(string, chan<- notify.EventInfo, ...notify.Event) error {
					return fmt.Errorf("failed to start watch trigger")
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := StartTrigger(context.Background(), test.trigger)
			time.Sleep(1 * time.Second)
			t.CheckNoError(err)
		})
	}
}
