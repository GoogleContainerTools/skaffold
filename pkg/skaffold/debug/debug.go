/*
Copyright 2021 The Skaffold Authors

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

package debug

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
)

type Config interface {
	Mode() config.RunMode
}

// Debugger defines the behavior for any implementation of a component
// that attaches to and helps debug deployed resources from Skaffold.
type Debugger interface {
	// Start starts the debugger.
	Start(context.Context) error

	// Stop stops the debugger.
	Stop()

	// Name returns an identifier string for the debugger.
	Name() string
}

type NoopDebugger struct{}

func (n *NoopDebugger) Start(context.Context) error { return nil }

func (n *NoopDebugger) Stop() {}

func (n *NoopDebugger) Name() string { return "Noop Debugger" }
