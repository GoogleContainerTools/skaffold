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

type LoggerMux []Logger

func (l LoggerMux) Start(ctx context.Context, out io.Writer) error {
	for _, logger := range l {
		if err := logger.Start(ctx, out); err != nil {
			return err
		}
	}
	return nil
}

func (l LoggerMux) Stop() {
	for _, logger := range l {
		logger.Stop()
	}
}

func (l LoggerMux) Mute() {
	for _, logger := range l {
		logger.Mute()
	}
}

func (l LoggerMux) Unmute() {
	for _, logger := range l {
		logger.Unmute()
	}
}

func (l LoggerMux) SetSince(t time.Time) {
	for _, logger := range l {
		logger.SetSince(t)
	}
}

func (l LoggerMux) RegisterArtifacts(artifacts []graph.Artifact) {
	for _, logger := range l {
		logger.RegisterArtifacts(artifacts)
	}
}
