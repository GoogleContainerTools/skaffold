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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resource"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

type Resource interface {

	// Name returns resource Name
	Name() string

	// Status returns resource status
	Status() resource.Status

	// ReportSinceLastUpdated returns the resource status only if it was updated
	// since last time reported.
	ReportSinceLastUpdated() string

	// UpdateStatus updates the resource status
	UpdateStatus(string, error)

	// IsStatusCheckComplete returns if the resource status check is complele
	IsStatusCheckComplete() bool

	// Deadline returns the deadline for the resource
	Deadline() time.Duration

	// CheckStatus checks resource status
	CheckStatus(context.Context, *runcontext.RunContext)
}
