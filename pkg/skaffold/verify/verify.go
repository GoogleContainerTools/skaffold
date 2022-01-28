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

package verify

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
)

// Verifier is the Verify API of skaffold and responsible for deploying
// the build results to a Kubernetes cluster
type Verifier interface {
	// Verify should ensure that the verify test is run to completion
	Verify(context.Context, io.Writer, []graph.Artifact) error

	// Dependencies returns a list of files that the deployer depends on.
	// In dev mode, a redeploy will be triggered
	Dependencies() ([]string, error)

	// Cleanup deletes what was deployed/executed by calling Verify.
	Cleanup(context.Context, io.Writer, bool) error

	// GetLogger returns a Verifier's implementation of a Logger
	GetLogger() log.Logger

	// GetAccessor returns a Verifier's implementation of an Accessor
	GetAccessor() access.Accessor

	// TODO(aaron-prindle) check whether it makes sense to keep Syncer in the verify API
	// GetSyncer returns a Verifier's implementation of a Syncer
	GetSyncer() sync.Syncer

	// TrackBuildArtifacts registers build artifacts to be tracked by a Verifier
	TrackBuildArtifacts([]graph.Artifact)

	// RegisterLocalImages tracks all local images to be loaded by the Verifier
	RegisterLocalImages([]graph.Artifact)

	// GetStatusMonitor returns a Verifier's implementation of a StatusMonitor
	GetStatusMonitor() status.Monitor
}
