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
	"context"
	"fmt"
	"io"
)

// ActionInvoker is the minimal interface required by the hooks package to
// dispatch an `ActionHook` to the configured custom-actions runner. It is
// injected at deployer-construction time to avoid a package cycle with
// pkg/skaffold/actions.
type ActionInvoker interface {
	// Invoke runs the named custom action synchronously, writing the
	// action's combined output to out. Implementations return a non-nil
	// error if the action cannot be found or fails.
	Invoke(ctx context.Context, out io.Writer, action string) error
}

// errNoActionInvoker is returned when a config declares an `action:` hook but
// the runtime has no actions runner wired in.
var errNoActionInvoker = fmt.Errorf("action hooks are not supported in this context: no custom-actions runner available")

// defaultActionInvoker is the process-wide invoker populated by
// SetDefaultActionInvoker at runner-construction time. Deploy hook runners
// built via newDeployRunner / newCloudRunDeployRunner read from this variable
// rather than receiving the invoker through every deployer's constructor,
// which would otherwise require threading it through many signatures.
//
// The variable is written exactly once per process (from runner.New, after
// GetActionsRunner succeeds) and read from hook goroutines. Writing once at
// startup and reading afterwards is safe under Go's happens-before rules.
var defaultActionInvoker ActionInvoker

// SetDefaultActionInvoker registers the process-wide ActionInvoker used by
// deploy hook runners to dispatch `action:` hooks. Passing nil clears it.
func SetDefaultActionInvoker(inv ActionInvoker) { defaultActionInvoker = inv }

func runActionHook(ctx context.Context, out io.Writer, inv ActionInvoker, name string) error {
	if inv == nil {
		return errNoActionInvoker
	}
	return inv.Invoke(ctx, out, name)
}
