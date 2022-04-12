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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	fsNotify "github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger/fsnotify"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewTrigger(t *testing.T) {
	tests := []struct {
		description       string
		trigger           string
		watchPollInterval int
		expected          Trigger
		shouldErr         bool
	}{
		{
			description:       "polling trigger",
			trigger:           "polling",
			watchPollInterval: 1,
			expected: &pollTrigger{
				Interval: 1 * time.Millisecond,
			},
		},
		{
			description:       "notify trigger",
			trigger:           "notify",
			watchPollInterval: 1,
			expected: fsNotify.New(map[string]struct{}{
				"../workspace":            {},
				"../some/other/workspace": {}}, nil, 1),
		},
		{
			description: "manual trigger",
			trigger:     "manual",
			expected:    &manualTrigger{},
		},
		{
			description: "unknown trigger",
			trigger:     "unknown",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := &mockConfig{
				trigger:           test.trigger,
				watchPollInterval: test.watchPollInterval,
				artifacts: []*latest.Artifact{
					{Workspace: "../workspace"},
					{Workspace: "../workspace"},
					{Workspace: "../some/other/workspace"},
				},
			}

			got, err := NewTrigger(cfg, nil)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expected, got, cmp.AllowUnexported(fsNotify.Trigger{}), cmp.Comparer(ignoreFuncComparer), cmp.AllowUnexported(manualTrigger{}), cmp.AllowUnexported(pollTrigger{}))
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
		mockWatch   func(string, chan<- notify.EventInfo, ...notify.Event) error
	}{
		{
			description: "fsNotify trigger works",
			mockWatch: func(string, chan<- notify.EventInfo, ...notify.Event) error {
				return nil
			},
		},
		{
			description: "fallback on polling trigger",
			mockWatch: func(string, chan<- notify.EventInfo, ...notify.Event) error {
				return fmt.Errorf("failed to start watch trigger")
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&fsNotify.Watch, test.mockWatch)
			trigger := fsNotify.New(nil, func() bool { return false }, 1)
			_, err := StartTrigger(context.Background(), trigger)
			time.Sleep(1 * time.Second)
			t.CheckNoError(err)
		})
	}
}

type mockConfig struct {
	trigger           string
	watchPollInterval int
	artifacts         []*latest.Artifact
}

func (c *mockConfig) Trigger() string               { return c.trigger }
func (c *mockConfig) WatchPollInterval() int        { return c.watchPollInterval }
func (c *mockConfig) Artifacts() []*latest.Artifact { return c.artifacts }
