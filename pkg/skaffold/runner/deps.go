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

package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DependenciesForArtifact lists the dependencies for a given artifact.
func DependenciesForArtifact(ctx context.Context, a *latest.Artifact) ([]string, error) {
	// The files to watch are overridden for this artifact.
	if len(a.WatchIncludes) > 0 || len(a.WatchExcludes) > 0 {
		files, err := util.ListFiles(a.Workspace, a.WatchIncludes, a.WatchExcludes)
		if err != nil {
			return nil, err
		}

		var paths []string
		for file := range files {
			paths = append(paths, filepath.Join(a.Workspace, file))
		}
		sort.Strings(paths)
		logrus.Debugf("Found dependencies: %v", paths)

		return paths, nil
	}

	var (
		paths []string
		err   error
	)

	switch {
	case a.DockerArtifact != nil:
		paths, err = docker.GetDependencies(ctx, a.Workspace, a.DockerArtifact)

	case a.BazelArtifact != nil:
		paths, err = bazel.GetDependencies(ctx, a.Workspace, a.BazelArtifact)

	case a.JibMavenArtifact != nil:
		paths, err = jib.GetDependenciesMaven(ctx, a.Workspace, a.JibMavenArtifact)

	case a.JibGradleArtifact != nil:
		paths, err = jib.GetDependenciesGradle(ctx, a.Workspace, a.JibGradleArtifact)

	default:
		return nil, fmt.Errorf("undefined artifact type: %+v", a.ArtifactType)
	}

	if err != nil {
		// if the context was cancelled act as if all is well
		// TODO(dgageot): this should be even higher in the call chain.
		if ctx.Err() == context.Canceled {
			logrus.Debugln(errors.Wrap(err, "ignore error since context is cancelled"))
			return nil, nil
		}

		return nil, err
	}

	var p []string
	for _, path := range paths {
		// TODO(dgageot): this is only done for jib builder.
		if !filepath.IsAbs(path) {
			path = filepath.Join(a.Workspace, path)
		}
		p = append(p, path)
	}
	return p, nil
}
