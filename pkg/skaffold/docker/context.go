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

package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
)

// BuildContext creates an archive of the build context to be sent to `docker build`.
// This code is mostly copied from docker/cli codebase:
// https://github.com/docker/cli/blob/ae66898200af606f900face29c3c6e8a738a1f40/cli/command/image/build.go#L228
func BuildContext(workspace, dockerfilePath string) (io.ReadCloser, string, error) {
	absDockerfile, err := NormalizeDockerfilePath(workspace, dockerfilePath)
	if err != nil {
		return nil, "", fmt.Errorf("normalizing dockerfile path: %w", err)
	}

	contextDir, relDockerfile, err := build.GetContextFromLocalDir(workspace, absDockerfile)
	if err != nil {
		return nil, "", fmt.Errorf("unable to prepare context: %w", err)
	}

	var dockerfileCtx io.ReadCloser
	if strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
		dockerfileCtx, err = os.Open(absDockerfile)
		if err != nil {
			return nil, "", fmt.Errorf("unable to open Dockerfile: %w", err)
		}
		defer dockerfileCtx.Close()
	}

	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return nil, "", err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return nil, "", fmt.Errorf("error checking context: %w", err)
	}

	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)

	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return nil, "", fmt.Errorf("can't create tar from folder %q: %w", contextDir, err)
	}

	// replace Dockerfile if it was added from a file outside the build-context.
	if dockerfileCtx != nil {
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return nil, "", err
		}
	}

	return buildCtx, relDockerfile, nil
}
