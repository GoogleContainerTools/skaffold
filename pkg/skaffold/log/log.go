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

package log

import (
	"context"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
)

// Logger defines the behavior of the object that retrieves logs from deployed resources.
// Logger implementations are platform-specific, and are controlled by a single Deployer.
type Logger interface {
	// Start starts the logger.
	Start(context.Context, io.Writer) error

	// Stop stops the logger.
	Stop()

	// Mute mutes the logger.
	Mute()

	// Unmute unmutes the logger.
	Unmute()

	// SetSince sets the original timestamp for the logger.
	SetSince(time.Time)

	// RegisterArtifacts tracks build artifacts inside of a logger.
	// The logger sometimes uses information about the currently deployed artifacts
	// to actually retrieve logs (e.g. the Kubernetes PodSelector). Thus, we need to
	// track the current build artifacts in the logger.
	RegisterArtifacts([]graph.Artifact)
}

// NoopLogger is used in tests. It will never retrieve any logs from any resources.
type NoopLogger struct{}

func (n *NoopLogger) Start(context.Context, io.Writer) error { return nil }

func (n *NoopLogger) Stop() {}

func (n *NoopLogger) Mute() {}

func (n *NoopLogger) Unmute() {}

func (n *NoopLogger) SetSince(time.Time) {}

func (n *NoopLogger) RegisterArtifacts(_ []graph.Artifact) {}
