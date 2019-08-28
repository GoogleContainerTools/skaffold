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

package deploy

import (
	"context"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resources"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

type Resource interface {
	// String returns the string representation of a resource.
	String() string
	// Type returns the type of the resource
	Type() string
	// Status returns the resource status
	Status() *resources.Status
	// Namespace returns the resource namespace
	Namespace() string
	// Name returns the resource name
	Name() string
	// CheckStatus performs the resource status check.
	CheckStatus(ctx context.Context, runCtx *runcontext.RunContext)
	// Deadline returns the status check deadline for the resource
	Deadline() time.Duration
	// UpdateStatus updates the status of a resource with the given error message
	UpdateStatus(msg string, reason string, err error)
	// IsStatusCheckComplete returns if a status check is complete for a resource
	IsStatusCheckComplete() bool
	// ReportSinceLastUpdated writes the last known status to out if it hasn't been reported earlier.
	ReportSinceLastUpdated(out io.Writer)
}
