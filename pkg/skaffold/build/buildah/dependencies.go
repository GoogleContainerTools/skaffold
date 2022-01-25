package buildah

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// GetDependencies finds the sources dependency for the given buildah artifact.
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, workspace string, a *latestV1.BuildahArtifact, cfg docker.Config) ([]string, error) {
	absContainerfile, err := docker.NormalizeDockerfilePath(workspace, a.ContainerFilePath)
	if err != nil {
		return nil, fmt.Errorf("normalizing containerfile path: %w", err)
	}

	buildArgs := make(map[string]*string, len(a.BuildArgs))
	for key, val := range a.BuildArgs {
		val := val
		newVal := &val
		buildArgs[key] = newVal
	}

	fts, err := docker.ReadCopyCmdsFromDockerfile(ctx, false, absContainerfile, workspace, buildArgs, cfg)
	if err != nil {
		return nil, fmt.Errorf("reading copy cmds from dockerfile: %w", err)
	}

	excludes, err := docker.ReadDockerignore(workspace, absContainerfile)
	if err != nil {
		return nil, fmt.Errorf("reading dockerignore: %w", err)
	}

	deps := make([]string, 0, len(fts))
	for _, ft := range fts {
		deps = append(deps, ft.From)
	}

	files, err := docker.WalkWorkspace(workspace, excludes, deps)
	if err != nil {
		return nil, fmt.Errorf("walking workspace: %w", err)
	}

	// Always add dockerfile even if it's .dockerignored. The daemon will need it anyways.
	files[absContainerfile] = true

	// Ignore .dockerignore
	delete(files, ".dockerignore")

	var dependencies []string
	for file := range files {
		dependencies = append(dependencies, file)
	}

	return dependencies, nil
}
