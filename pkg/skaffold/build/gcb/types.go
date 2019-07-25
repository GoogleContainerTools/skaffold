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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
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
	skipTests          bool
	insecureRegistries map[string]bool
}

// NewBuilder creates a new Builder that builds artifacts with Google Cloud Build.
func NewBuilder(runCtx *runcontext.RunContext) *Builder {
	return &Builder{
		GoogleCloudBuild:   runCtx.Cfg.Build.GoogleCloudBuild,
		skipTests:          runCtx.Opts.SkipTests,
		insecureRegistries: runCtx.InsecureRegistries,
	}
}

// Labels are labels specific to Google Cloud Build.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "google-cloud-build",
	}
}

// DependenciesForArtifact returns the dependencies for this artifact
func (b *Builder) DependenciesForArtifact(ctx context.Context, a *latest.Artifact) ([]string, error) {
	var paths []string
	var err error
	if a.DockerArtifact != nil {
		paths, err = docker.GetDependencies(ctx, a.Workspace, a.DockerArtifact.DockerfilePath, a.DockerArtifact.BuildArgs, b.insecureRegistries)
		if err != nil {
			return nil, errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
		}
	}
	if a.JibMavenArtifact != nil {
		paths, err = jib.GetDependenciesMaven(ctx, a.Workspace, a.JibMavenArtifact)
		if err != nil {
			return nil, errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
		}
	}
	if a.JibGradleArtifact != nil {
		paths, err = jib.GetDependenciesGradle(ctx, a.Workspace, a.JibGradleArtifact)
		if err != nil {
			return nil, errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
		}
	}

	return util.AbsolutePaths(a.Workspace, paths), nil
}

func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	return nil // noop
}

func (b *Builder) SyncMap(ctx context.Context, artifact *latest.Artifact) (map[string][]string, error) {
	return nil, build.ErrSyncMapNotSupported{}
}
