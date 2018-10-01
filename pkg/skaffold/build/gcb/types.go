/*
Copyright 2018 The Skaffold Authors

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

package gcb

import (
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	latest "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
)

const (
	// StatusUnknown "STATUS_UNKNOWN" - Status of the build is unknown.
	StatusUnknown = "STATUS_UNKNOWN"

	// StatusQueued "QUEUED" - Build is queued; work has not yet begun.
	StatusQueued = "QUEUED"

	// StatusWorking "WORKING" - Build is being executed.
	StatusWorking = "WORKING"

	// StatusSuccess  "SUCCESS" - Build finished successfully.
	StatusSuccess = "SUCCESS"

	// StatusFailure  "FAILURE" - Build failed to complete successfully.
	StatusFailure = "FAILURE"

	// StatusInternalError  "INTERNAL_ERROR" - Build failed due to an internal cause.
	StatusInternalError = "INTERNAL_ERROR"

	// StatusTimeout  "TIMEOUT" - Build took longer than was allowed.
	StatusTimeout = "TIMEOUT"

	// StatusCancelled  "CANCELLED" - Build was canceled by a user.
	StatusCancelled = "CANCELLED"

	// RetryDelay is the time to wait in between polling the status of the cloud build
	RetryDelay = 1 * time.Second
)

// Builder builds artifacts with Google Cloud Build.
type Builder struct {
	*latest.GoogleCloudBuild
}

// NewBuilder creates a new Builder that builds artifacts with Google Cloud Build.
func NewBuilder(cfg *latest.GoogleCloudBuild) *Builder {
	return &Builder{
		GoogleCloudBuild: cfg,
	}
}

// Labels are labels specific to Google Cloud Build.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "google-cloud-build",
	}
}
