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
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
)

// ExecStrategy represents the functions to use to execute a list of tasks.
type ExecStrategy func(ctx context.Context, out io.Writer, ts []Task) error

// Task represents a single container from a custom action.
type Task interface {
	// Name is the unique name of the tasks across all other tasks and actions.
	Name() string

	// Timeout is the max time allowed for the execution of this task.
	Timeout() time.Duration

	// Exec triggers the execution of the task.
	Exec(ctx context.Context, out io.Writer) error

	// Cleanup frees the resources created by the task to execute itself.
	Cleanup(ctx context.Context, out io.Writer) error
}

// Action represents a single custom action.
// It is composed by Tasks, and itself is a Task too.
type Action interface {
	Task

	// IsFailFast returns true if the execution strategy for the action is
	// fail-fast.
	IsFailFast() bool

	// SetExecFunc sets a function to be use to execute the tasks associated
	// with the action.
	SetExecFunc(f ExecStrategy)
}

// ExecEnv represents every execution mode available for custom actions.
type ExecEnv interface {
	// PrepareActions creates the shared resources needed for the actions of an
	// specific execution mode. It returns the actions of this execution mode.
	PrepareActions(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, acsNames []string) ([]Action, error)

	// Cleanup frees the shared resources created during PrepareActions.
	Cleanup(ctx context.Context, out io.Writer) error
}
