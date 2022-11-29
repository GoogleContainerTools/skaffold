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

package gcb

import (
	"context"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
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

	// BackoffFactor is the exponent for exponential backoff during build status polling
	BackoffFactor = 1.5

	// BackoffSteps is the number of times we increase the backoff time during exponential backoff
	BackoffSteps = 10

	// RetryTimeout is the max amount of time to retry getting the status of the build before erroring
	RetryTimeout = 3 * time.Minute
)

func NewStatusBackoff() *wait.Backoff {
	return &wait.Backoff{
		Duration: RetryDelay,
		Factor:   float64(BackoffFactor),
		Steps:    BackoffSteps,
		Cap:      60 * time.Second,
	}
}

// Builder builds artifacts with Google Cloud Build.
type Builder struct {
	*latest.GoogleCloudBuild

	cfg                Config
	skipTests          bool
	muted              build.Muted
	artifactStore      build.ArtifactStore
	sourceDependencies graph.SourceDependenciesCache
}

type Config interface {
	docker.Config

	SkipTests() bool
	Muted() config.Muted
}

type BuilderContext interface {
	Config
	ArtifactStore() build.ArtifactStore
	SourceDependenciesResolver() graph.SourceDependenciesCache
}

// NewBuilder creates a new Builder that builds artifacts with Google Cloud Build.
func NewBuilder(bCtx BuilderContext, buildCfg *latest.GoogleCloudBuild) *Builder {
	return &Builder{
		GoogleCloudBuild:   buildCfg,
		cfg:                bCtx,
		skipTests:          bCtx.SkipTests(),
		muted:              bCtx.Muted(),
		artifactStore:      bCtx.ArtifactStore(),
		sourceDependencies: bCtx.SourceDependenciesResolver(),
	}
}

func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return nil // noop
}

func (b *Builder) PushImages() bool {
	return true
}

func (b *Builder) SupportedPlatforms() platform.Matcher {
	return platform.All
}
