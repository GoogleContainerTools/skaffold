/*
Copyright 2023 The Skaffold Authors

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

package actions

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type mockTask struct {
	ExecF    func(ctx context.Context, out io.Writer, m *mockTask) error
	Finished bool
}

func (ma *mockTask) Name() string { return "" }

func (ma *mockTask) Timeout() time.Duration { return time.Second * 60 }

func (ma *mockTask) Exec(ctx context.Context, out io.Writer) error {
	if ma.ExecF != nil {
		return ma.ExecF(ctx, out, ma)
	}
	return nil
}

func (ma *mockTask) Cleanup(ctx context.Context, out io.Writer) error { return nil }

type mockAction struct {
	ExecF    func(ctx context.Context, out io.Writer) error
	Tasks    []*mockTask
	ExecS    ExecStrategy
	FailFast bool
}

func (ma *mockAction) Name() string { return "" }

func (ma *mockAction) Timeout() time.Duration { return time.Second * 1 }

func (ma *mockAction) Exec(ctx context.Context, out io.Writer) error {
	if ma.ExecF != nil {
		return ma.ExecF(ctx, out)
	}
	tasks := []Task{}
	for _, t := range ma.Tasks {
		tasks = append(tasks, t)
	}

	return ma.ExecS(ctx, out, tasks)
}

func (ma *mockAction) Cleanup(ctx context.Context, out io.Writer) error { return nil }

func (ma *mockAction) IsFailFast() bool { return ma.FailFast }

func (ma *mockAction) SetExecFunc(f ExecStrategy) { ma.ExecS = f }

type mockExecEnv struct {
	Actions     []string
	MockPrepAcs func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error)
	MockActions []*mockAction
}

func (me mockExecEnv) PrepareActions(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error) {
	if me.MockPrepAcs != nil {
		return me.MockPrepAcs(ctx, out, allbuilds, acsNames)
	}
	var acs []Action
	for _, a := range me.MockActions {
		acs = append(acs, a)
	}
	return acs, nil
}

func (me mockExecEnv) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func newRunner(mExecEnvs []mockExecEnv) Runner {
	execEnvByAction := map[string]ExecEnv{}
	acsByExecEnv := map[ExecEnv][]string{}
	execEnvs := []ExecEnv{}

	for _, execEnv := range mExecEnvs {
		for _, a := range execEnv.Actions {
			execEnvByAction[a] = &execEnv
			acsByExecEnv[&execEnv] = append(acsByExecEnv[&execEnv], a)
		}
		execEnvs = append(execEnvs, execEnv)
	}

	return NewRunner(execEnvByAction, execEnvs, acsByExecEnv)
}

func TestActionsRunner_Exec(t *testing.T) {
	tests := []struct {
		description  string
		actionToExec string
		shouldErr    bool
		err          string
		execEnvs     []mockExecEnv
	}{
		{
			description:  "action not found",
			actionToExec: "action3",
			shouldErr:    true,
			err:          "custom action action3 not found",
			execEnvs: []mockExecEnv{
				{
					Actions: []string{"action1", "action2"},
				},
			},
		},
		{
			description:  "zero actions returned",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "failed to create action1 action",
			execEnvs: []mockExecEnv{
				{
					Actions: []string{"action1", "action2"},
					MockPrepAcs: func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error) {
						return []Action{}, nil
					},
				},
			},
		},
		{
			description:  "more than one action returned",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "failed to create action1 action",
			execEnvs: []mockExecEnv{
				{
					Actions: []string{"action1", "action2"},
					MockPrepAcs: func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error) {
						return []Action{&mockAction{}, &mockAction{}}, nil
					},
				},
			},
		},
		{
			description:  "forward exec error",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "error from action execution",
			execEnvs: []mockExecEnv{
				{
					Actions: []string{"action1", "action2"},
					MockPrepAcs: func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error) {
						return []Action{&mockAction{
							ExecF: func(ctx context.Context, out io.Writer) error {
								return fmt.Errorf("error from action execution")
							},
						}}, nil
					},
				},
			},
		},
		{
			description:  "forward prepare actions error",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "error from prepare actions",
			execEnvs: []mockExecEnv{
				{
					Actions: []string{"action1", "action2"},
					MockPrepAcs: func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error) {
						return []Action{&mockAction{}}, fmt.Errorf("error from prepare actions")
					},
				},
			},
		},
		{
			description:  "exec successfully",
			actionToExec: "action1",
			execEnvs: []mockExecEnv{
				{
					Actions: []string{"action1", "action2"},
					MockPrepAcs: func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error) {
						return []Action{&mockAction{
							ExecF: func(ctx context.Context, out io.Writer) error { return nil },
						}}, nil
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runner := newRunner(test.execEnvs)
			err := runner.Exec(context.TODO(), nil, nil, test.actionToExec)

			if test.shouldErr {
				t.CheckErrorContains(test.err, err)
			} else {
				t.CheckNoError(err)
			}
		})
	}
}

func TestActionsRunner_ExecFailFast(t *testing.T) {
	tests := []struct {
		description  string
		actionToExec string
		shouldErr    bool
		err          string
		action       *mockAction
	}{
		{
			description:  "interrupt other actions when one fails",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "error from mock task",
			action: &mockAction{
				FailFast: true,
				Tasks: []*mockTask{
					{
						// This task will run for 10 secs, but will be interrupted when 2 seconds.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							case <-time.After(time.Second * 10):
								m.Finished = true
							case <-ctx.Done():
								m.Finished = false
								err = ctx.Err()
							}
							return err
						},
					},
					{
						// This task will fail first.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							time.Sleep(time.Second * 2)
							return fmt.Errorf("error from mock task")
						},
					},
				},
			},
		},
		{
			description:  "execute all actions till end",
			actionToExec: "action1",
			shouldErr:    false,
			action: &mockAction{
				FailFast: true,
				Tasks: []*mockTask{
					{
						// This task will run for 10 secs, but will be interrupted when 2 seconds.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							case <-time.After(time.Second * 5):
								m.Finished = true
							case <-ctx.Done():
								m.Finished = false
								err = ctx.Err()
							}
							return err
						},
					},
					{
						// This task will fail first.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							time.Sleep(time.Second * 2)
							m.Finished = true
							return nil
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {

			execEnvs := []mockExecEnv{
				{
					Actions: []string{test.actionToExec},
					MockActions: []*mockAction{
						test.action,
					},
				},
			}

			runner := newRunner(execEnvs)
			err := runner.Exec(context.TODO(), nil, nil, test.actionToExec)

			if test.shouldErr {
				t.CheckErrorContains(test.err, err)
				for _, task := range test.action.Tasks {
					t.CheckFalse(task.Finished)
				}
			} else {
				for _, task := range test.action.Tasks {
					t.CheckTrue(task.Finished)
				}
				t.CheckNoError(err)
			}
		})
	}
}

func TestActionsRunner_ExecFailSafe(t *testing.T) {
	tests := []struct {
		description  string
		actionToExec string
		shouldErr    bool
		err          string
		action       *mockAction
	}{
		{
			description:  "run all actions till end if one fails",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "error from mock task",
			action: &mockAction{
				FailFast: false,
				Tasks: []*mockTask{
					{
						// This task will run for 10 secs, but will be interrupted when 2 seconds.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							case <-time.After(time.Second * 10):
								m.Finished = true
							case <-ctx.Done():
								m.Finished = false
								err = ctx.Err()
							}
							return err
						},
					},
					{
						// This task will fail first.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							time.Sleep(time.Second * 2)
							m.Finished = true
							return fmt.Errorf("error from mock task")
						},
					},
				},
			},
		},
		{
			description:  "execute all actions till end",
			actionToExec: "action1",
			shouldErr:    false,
			action: &mockAction{
				FailFast: true,
				Tasks: []*mockTask{
					{
						// This task will run for 10 secs, but will be interrupted when 2 seconds.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							case <-time.After(time.Second * 5):
								m.Finished = true
							case <-ctx.Done():
								m.Finished = false
								err = ctx.Err()
							}
							return err
						},
					},
					{
						// This task will fail first.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							time.Sleep(time.Second * 2)
							m.Finished = true
							return nil
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {

			execEnvs := []mockExecEnv{
				{
					Actions: []string{test.actionToExec},
					MockActions: []*mockAction{
						test.action,
					},
				},
			}

			runner := newRunner(execEnvs)
			err := runner.Exec(context.TODO(), nil, nil, test.actionToExec)

			if test.shouldErr {
				t.CheckErrorContains(test.err, err)
			} else {
				t.CheckNoError(err)
			}
			for _, task := range test.action.Tasks {
				t.CheckTrue(task.Finished)
			}
		})
	}
}
