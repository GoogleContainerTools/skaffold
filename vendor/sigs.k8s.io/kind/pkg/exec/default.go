/*
Copyright 2019 The Kubernetes Authors.

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

package exec

import "context"

// DefaultCmder is a LocalCmder instance used for convenience, packages
// originally using os/exec.Command can instead use pkg/kind/exec.Command
// which forwards to this instance
// TODO(bentheelder): swap this for testing
// TODO(bentheelder): consider not using a global for this :^)
var DefaultCmder = &LocalCmder{}

// Command is a convenience wrapper over DefaultCmder.Command
func Command(command string, args ...string) Cmd {
	return DefaultCmder.Command(command, args...)
}

// CommandContext is a convenience wrapper over DefaultCmder.CommandContext
func CommandContext(ctx context.Context, command string, args ...string) Cmd {
	return DefaultCmder.CommandContext(ctx, command, args...)
}
