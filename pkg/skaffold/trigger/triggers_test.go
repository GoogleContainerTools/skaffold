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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

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

			got, err := NewTrigger(runCtx)
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expected, got, cmp.AllowUnexported(fsNotifyTrigger{}))
			}
		})
	}
}

func TestPollTrigger_Debounce(t *testing.T) {
	trigger := &pollTrigger{}
	got, want := trigger.Debounce(), true
	testutil.CheckDeepEqual(t, want, got)
}

func TestPollTrigger_LogWatchToUser(t *testing.T) {
	out := new(bytes.Buffer)

	trigger := &pollTrigger{Interval: 10}
	trigger.LogWatchToUser(out)

	got, want := out.String(), "Watching for changes every 10ns...\n"
	testutil.CheckDeepEqual(t, want, got)
}

func TestNotifyTrigger_Debounce(t *testing.T) {
	trigger := &fsNotifyTrigger{}
	got, want := trigger.Debounce(), false
	testutil.CheckDeepEqual(t, want, got)
}

func TestNotifyTrigger_LogWatchToUser(t *testing.T) {
	out := new(bytes.Buffer)

	trigger := &fsNotifyTrigger{Interval: 10}
	trigger.LogWatchToUser(out)

	got, want := out.String(), "Watching for changes...\n"
	testutil.CheckDeepEqual(t, want, got)
}

func TestManualTrigger_Debounce(t *testing.T) {
	trigger := &manualTrigger{}
	got, want := trigger.Debounce(), false
	testutil.CheckDeepEqual(t, want, got)
}

func TestManualTrigger_LogWatchToUser(t *testing.T) {
	out := new(bytes.Buffer)

	trigger := &manualTrigger{}
	trigger.LogWatchToUser(out)

	got, want := out.String(), "Press any key to rebuild/redeploy the changes\n"
	testutil.CheckDeepEqual(t, want, got)
}
