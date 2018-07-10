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

package watch

import (
	"context"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Factory is used to create Watchers.
type Factory func(files []string, artifacts []*v1alpha2.Artifact, pollInterval time.Duration, opts *config.SkaffoldOptions) CompositeWatcher

// CompositeWatcher can watch both files and artifacts.
type CompositeWatcher interface {
	Run(ctx context.Context, onFileChange FileChangedFn, onArtifactChange ArtifactChangedFn) error
}

type compositeWatcher struct {
	files        []string
	artifacts    []*v1alpha2.Artifact
	pollInterval time.Duration
	opts         *config.SkaffoldOptions
}

// NewCompositeWatcher creates a CompositeWatcher that watches both files and artifacts.
func NewCompositeWatcher(files []string, artifacts []*v1alpha2.Artifact, pollInterval time.Duration, opts *config.SkaffoldOptions) CompositeWatcher {
	return &compositeWatcher{
		files:        files,
		artifacts:    artifacts,
		pollInterval: pollInterval,
		opts:         opts,
	}
}

func (w *compositeWatcher) Run(ctx context.Context, onFileChange FileChangedFn, onArtifactChange ArtifactChangedFn) error {
	artifactWatcher, err := NewArtifactWatcher(w.artifacts, w.pollInterval, w.opts.GitRepository)
	if err != nil {
		return errors.Wrap(err, "watching artifacts")
	}

	fileWatcher, err := NewFileWatcher(w.files, w.pollInterval)
	if err != nil {
		return errors.Wrap(err, "watching files")
	}

	g, watchCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return artifactWatcher.Run(watchCtx, onArtifactChange)
	})
	g.Go(func() error {
		return fileWatcher.Run(watchCtx, onFileChange)
	})
	return g.Wait()
}
