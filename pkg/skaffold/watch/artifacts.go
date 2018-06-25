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
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ArtifactChangedFn is a function called when artifacts where changed.
type ArtifactChangedFn func(changes []*v1alpha2.Artifact) error

// ArtifactWatcher watches for artifacts changes.
type ArtifactWatcher interface {
	Run(ctx context.Context, callback ArtifactChangedFn) error
}

type artifactWatcher struct {
	artifacts   []*v1alpha2.Artifact
	fileWatcher FileWatcher
}

// NewArtifactWatcher creates an ArtifactWatcher for a list of artifacts.
func NewArtifactWatcher(artifacts []*v1alpha2.Artifact, pollInterval time.Duration) (ArtifactWatcher, error) {
	fileWatcher, err := NewFileWatcher(workingDirs(artifacts), pollInterval)
	if err != nil {
		return nil, errors.Wrap(err, "creating file watcher")
	}

	return &artifactWatcher{
		artifacts:   artifacts,
		fileWatcher: fileWatcher,
	}, nil
}

func workingDirs(artifacts []*v1alpha2.Artifact) []string {
	var workingDirs []string

	for _, artifact := range artifacts {
		workingDirs = append(workingDirs, artifact.Workspace)
	}

	return workingDirs
}

func (w *artifactWatcher) Run(ctx context.Context, callback ArtifactChangedFn) error {
	return w.fileWatcher.Run(ctx, func(changes []string) error {
		artifacts, err := w.artifactsForFiles(changes)
		if err != nil {
			logrus.Warnln("Skipping build. Dependencies couldn't be listed", err)
			return nil
		}

		if len(artifacts) == 0 {
			return nil
		}

		err = callback(artifacts)
		if err != nil {
			return errors.Wrap(err, "applying changes to artifacts")
		}

		return nil
	})
}

func (w *artifactWatcher) artifactsForFiles(files []string) ([]*v1alpha2.Artifact, error) {
	var artifacts []*v1alpha2.Artifact

	for _, artifact := range w.artifacts {
		matches, err := matchesAny(artifact, files)
		if err != nil {
			return nil, errors.Wrap(err, "matching path to artifact")
		}

		if matches {
			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts, nil
}

func matchesAny(artifact *v1alpha2.Artifact, files []string) (bool, error) {
	inWorkspace, err := inWorkspace(artifact, files)
	if err != nil {
		return false, errors.Wrap(err, "getting files in workspace")
	}

	if len(inWorkspace) == 0 {
		return false, nil
	}

	deps, err := build.DependenciesForArtifact(artifact)
	if err != nil {
		return false, errors.Wrap(err, "getting dependencies for artifact")
	}

	for _, path := range inWorkspace {
		for _, dep := range deps {
			if path == filepath.Join(artifact.Workspace, dep) {
				return true, nil
			}
		}
	}

	return false, nil
}

// inWorkspace filters out the paths that are not in the artifact's workspace.
func inWorkspace(artifact *v1alpha2.Artifact, files []string) ([]string, error) {
	var inWorkspace []string

	for _, file := range files {
		p1, err := filepath.Abs(file)
		if err != nil {
			return nil, errors.Wrapf(err, "getting absolute path for %s", file)
		}

		p2, _ := filepath.Abs(artifact.Workspace)
		if err != nil {
			return nil, errors.Wrapf(err, "getting absolute path for %s", artifact.Workspace)
		}

		if strings.HasPrefix(p1, p2) {
			inWorkspace = append(inWorkspace, file)
		}
	}

	return inWorkspace, nil
}
