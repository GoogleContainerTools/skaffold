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

package access

import (
	"context"
	"io"
)

// Accessor defines the behavior for any implementation of a component
// that accesses and exposes deployed resources from Skaffold.
type Accessor interface {
	// Start starts the resource accessor.
	Start(context.Context, io.Writer, []string) error

	// Stop stops the resource accessor.
	Stop()
}

type NoopAccessor struct{}

func (n *NoopAccessor) Start(context.Context, io.Writer, []string) error { return nil }

func (n *NoopAccessor) Stop() {}
