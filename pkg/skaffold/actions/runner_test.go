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
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

type mockTask struct {
	ExecF    func(ctx context.Context, out io.Writer, m *mockTask) error
	Finished bool
}

func (ma *mockTask) Name() string { return "" }

func (ma *mockTask) Exec(ctx context.Context, out io.Writer) error {
	if ma.ExecF != nil {
		return ma.ExecF(ctx, out, ma)
	}
	ma.Finished = true
	return nil
}

func (ma *mockTask) Cleanup(ctx context.Context, out io.Writer) error { return nil }

type mockExecEnv struct {
	Actions     []string
	MockPrepAcs func(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error)
	MockActions []Action
}

func (me mockExecEnv) PrepareActions(ctx context.Context, out io.Writer, allbuilds, localImgs []graph.Artifact, acsNames []string) ([]Action, error) {
	if me.MockPrepAcs != nil {
		return me.MockPrepAcs(ctx, out, allbuilds, acsNames)
	}
	return me.MockActions, nil
}

func (me mockExecEnv) Stop() {}

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
						return []Action{{}, {}}, nil
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
						return []Action{
							{
								execFunc: func(ctx context.Context, out io.Writer, ts []Task) error {
									return fmt.Errorf("error from action execution")
								},
							},
						}, nil
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
						return nil, fmt.Errorf("error from prepare actions")
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
						return []Action{
							{
								execFunc: func(ctx context.Context, out io.Writer, ts []Task) error {
									return nil
								},
							},
						}, nil
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := new(bytes.Buffer)
			runner := newRunner(test.execEnvs)
			err := runner.Exec(context.TODO(), out, nil, nil, test.actionToExec)

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
		action       Action
	}{
		{
			description:  "interrupt other tasks when one fails",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "error from mock task",
			action: Action{
				tasks: []Task{
					&mockTask{
						// This task will fail, interrupting the others.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							return fmt.Errorf("error from mock task")
						},
					},
					&mockTask{
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							// This task will run for 10 secs, but should be interrupted
							// in 2 seconds due to the other task failed, so this never finish.
							case <-time.After(time.Second * 5):
								m.Finished = true
							case <-ctx.Done():
								err = ctx.Err()
							}
							return err
						},
					},
				},
			},
		},
		{
			description:  "interrupt all tasks when on action timeout",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "context deadline",
			action: Action{
				timeout: 1,
				tasks: []Task{
					&mockTask{
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							case <-time.After(time.Second * 2):
								m.Finished = true
							case <-ctx.Done():
								err = ctx.Err()
							}
							return err
						},
					},
					&mockTask{
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = false
							var err error
							select {
							case <-time.After(time.Second * 2):
								m.Finished = true
							case <-ctx.Done():
								err = ctx.Err()
							}
							return err
						},
					},
				},
			},
		},
		{
			description:  "execute all tasks till end",
			actionToExec: "action1",
			shouldErr:    false,
			action: Action{
				tasks: []Task{&mockTask{}, &mockTask{}},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testEvent.InitializeState([]latest.Pipeline{{}})
			out := new(bytes.Buffer)
			test.action.execFunc = execWithFailingFast
			execEnvs := []mockExecEnv{
				{
					Actions:     []string{test.actionToExec},
					MockActions: []Action{test.action},
				},
			}

			runner := newRunner(execEnvs)
			err := runner.Exec(context.TODO(), out, nil, nil, test.actionToExec)

			if test.shouldErr {
				t.CheckErrorContains(test.err, err)
				for _, task := range test.action.tasks {
					t.CheckFalse(task.(*mockTask).Finished)
				}
			} else {
				for _, task := range test.action.tasks {
					t.CheckTrue(task.(*mockTask).Finished)
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
		action       Action
	}{
		{
			description:  "run all tasks till end if one fails",
			actionToExec: "action1",
			shouldErr:    true,
			err:          "error from mock task",
			action: Action{
				tasks: []Task{
					&mockTask{},
					&mockTask{
						// This task will fail first.
						ExecF: func(ctx context.Context, out io.Writer, m *mockTask) error {
							m.Finished = true
							return fmt.Errorf("error from mock task")
						},
					},
				},
			},
		},
		{
			description:  "execute all tasks till end",
			actionToExec: "action1",
			shouldErr:    false,
			action: Action{
				tasks: []Task{&mockTask{}, &mockTask{}},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testEvent.InitializeState([]latest.Pipeline{{}})
			out := new(bytes.Buffer)
			test.action.execFunc = execWithFailingFast
			execEnvs := []mockExecEnv{
				{
					Actions:     []string{test.actionToExec},
					MockActions: []Action{test.action},
				},
			}

			runner := newRunner(execEnvs)
			err := runner.Exec(context.TODO(), out, nil, nil, test.actionToExec)

			if test.shouldErr {
				t.CheckErrorContains(test.err, err)
			} else {
				t.CheckNoError(err)
			}
			for _, task := range test.action.tasks {
				t.CheckTrue(task.(*mockTask).Finished)
			}
		})
	}
}
