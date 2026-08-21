/*
Copyright 2026 The Skaffold Authors

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

package hooks

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type fakeActionInvoker struct {
	calls []string
	err   map[string]error
	out   string
}

func (f *fakeActionInvoker) Invoke(_ context.Context, out io.Writer, action string) error {
	f.calls = append(f.calls, action)
	if f.out != "" {
		_, _ = out.Write([]byte(f.out))
	}
	if e, ok := f.err[action]; ok {
		return e
	}
	return nil
}

// installInvoker registers fake as the process-wide invoker for the duration
// of the test and restores any previous value on cleanup.
func installInvoker(t *testing.T, fake ActionInvoker) {
	t.Helper()
	prev := defaultActionInvoker
	SetDefaultActionInvoker(fake)
	t.Cleanup(func() { SetDefaultActionInvoker(prev) })
}

func TestDeployActionHook_Dispatch(t *testing.T) {
	testutil.Run(t, "invokes pre then post in order", func(t *testutil.T) {
		fake := &fakeActionInvoker{out: "action-output"}
		installInvoker(t.T, fake)

		h := latest.DeployHooks{
			PreHooks: []latest.DeployHookItem{
				{ActionHook: &latest.ActionHook{Name: "pre-one"}},
				{ActionHook: &latest.ActionHook{Name: "pre-two"}},
			},
			PostHooks: []latest.DeployHookItem{
				{ActionHook: &latest.ActionHook{Name: "post-one"}},
			},
		}

		opts := NewDeployEnvOpts("run_id", "ctx", []string{"ns"})
		ns := []string{"ns"}
		runner := NewDeployRunner(nil, h, &ns, nil, opts, nil)

		var out bytes.Buffer
		t.CheckNoError(runner.RunPreHooks(context.Background(), &out))
		t.CheckNoError(runner.RunPostHooks(context.Background(), &out))
		t.CheckDeepEqual([]string{"pre-one", "pre-two", "post-one"}, fake.calls)
		t.CheckContains("pre-deploy action \"pre-one\"", out.String())
		t.CheckContains("post-deploy action \"post-one\"", out.String())
		t.CheckContains("action-output", out.String())
	})
}

func TestDeployActionHook_ErrorStopsOrdering(t *testing.T) {
	testutil.Run(t, "first failure aborts subsequent hooks", func(t *testutil.T) {
		fake := &fakeActionInvoker{err: map[string]error{"bad": errors.New("boom")}}
		installInvoker(t.T, fake)

		h := latest.DeployHooks{
			PreHooks: []latest.DeployHookItem{
				{ActionHook: &latest.ActionHook{Name: "ok"}},
				{ActionHook: &latest.ActionHook{Name: "bad"}},
				{ActionHook: &latest.ActionHook{Name: "never"}},
			},
		}
		opts := NewDeployEnvOpts("run_id", "ctx", nil)
		ns := []string{}
		runner := NewDeployRunner(nil, h, &ns, nil, opts, nil)

		var out bytes.Buffer
		err := runner.RunPreHooks(context.Background(), &out)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		t.CheckContains("pre-deploy action \"bad\"", err.Error())
		t.CheckContains("boom", err.Error())
		t.CheckDeepEqual([]string{"ok", "bad"}, fake.calls)
	})
}

func TestDeployActionHook_NoInvokerRegistered(t *testing.T) {
	testutil.Run(t, "missing invoker returns errNoActionInvoker", func(t *testutil.T) {
		installInvoker(t.T, nil)

		h := latest.DeployHooks{
			PreHooks: []latest.DeployHookItem{
				{ActionHook: &latest.ActionHook{Name: "x"}},
			},
		}
		opts := NewDeployEnvOpts("run_id", "ctx", nil)
		ns := []string{}
		runner := NewDeployRunner(nil, h, &ns, nil, opts, nil)

		var out bytes.Buffer
		err := runner.RunPreHooks(context.Background(), &out)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		t.CheckContains("no custom-actions runner available", err.Error())
	})
}
