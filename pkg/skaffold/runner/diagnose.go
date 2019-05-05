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

package runner

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
)

func (r *SkaffoldRunner) DiagnoseArtifacts(out io.Writer) error {
	ctx := context.Background()

	for _, artifact := range r.runCtx.Cfg.Build.Artifacts {
		color.Default.Fprintf(out, "\n%s: %s\n", typeOfArtifact(artifact), artifact.ImageName)

		if artifact.DockerArtifact != nil {
			size, err := sizeOfDockerContext(ctx, artifact, r.runCtx.InsecureRegistries)
			if err != nil {
				return errors.Wrap(err, "computing the size of the Docker context")
			}

			fmt.Fprintf(out, " - Size of the context: %vbytes\n", size)
		}

		timeDeps1, deps, err := timeToListDependencies(ctx, r.Builder, artifact)
		if err != nil {
			return errors.Wrap(err, "listing artifact dependencies")
		}
		timeDeps2, _, err := timeToListDependencies(ctx, r.Builder, artifact)
		if err != nil {
			return errors.Wrap(err, "listing artifact dependencies")
		}

		fmt.Fprintln(out, " - Dependencies:", len(deps), "files")
		fmt.Fprintf(out, " - Time to list dependencies: %v (2nd time: %v)\n", timeDeps1, timeDeps2)

		timeSyncMap1, err := timeToConstructSyncMap(ctx, r.Builder, artifact)
		if err != nil {
			if _, isNotSupported := err.(build.ErrSyncMapNotSupported); !isNotSupported {
				return errors.Wrap(err, "construct artifact dependencies")
			}
		}
		timeSyncMap2, err := timeToConstructSyncMap(ctx, r.Builder, artifact)
		if err != nil {
			if _, isNotSupported := err.(build.ErrSyncMapNotSupported); !isNotSupported {
				return errors.Wrap(err, "construct artifact dependencies")
			}
		} else {
			fmt.Fprintf(out, " - Time to construct sync-map: %v (2nd time: %v)\n", timeSyncMap1, timeSyncMap2)
		}

		timeMTimes1, err := timeToComputeMTimes(deps)
		if err != nil {
			return errors.Wrap(err, "computing modTimes")
		}
		timeMTimes2, err := timeToComputeMTimes(deps)
		if err != nil {
			return errors.Wrap(err, "computing modTimes")
		}

		fmt.Fprintf(out, " - Time to compute mTimes on dependencies: %v (2nd time: %v)\n", timeMTimes1, timeMTimes2)
	}

	return nil
}

func typeOfArtifact(a *latest.Artifact) string {
	switch {
	case a.DockerArtifact != nil:
		return "Docker artifact"
	case a.BazelArtifact != nil:
		return "Bazel artifact"
	case a.JibGradleArtifact != nil:
		return "Jib Gradle artifact"
	case a.JibMavenArtifact != nil:
		return "Jib Maven artifact"
	default:
		return "Unknown artifact"
	}
}

func timeToListDependencies(ctx context.Context, builder build.Builder, a *latest.Artifact) (time.Duration, []string, error) {
	start := time.Now()
	paths, err := builder.DependenciesForArtifact(ctx, a)
	return time.Since(start), paths, err
}

func timeToConstructSyncMap(ctx context.Context, builder build.Builder, a *latest.Artifact) (time.Duration, error) {
	start := time.Now()
	_, err := builder.SyncMap(ctx, a)
	return time.Since(start), err
}

func timeToComputeMTimes(deps []string) (time.Duration, error) {
	start := time.Now()

	if _, err := watch.Stat(func() ([]string, error) { return deps, nil }); err != nil {
		return 0, errors.Wrap(err, "computing modTimes")
	}

	return time.Since(start), nil
}

func sizeOfDockerContext(ctx context.Context, a *latest.Artifact, insecureRegistries map[string]bool) (int64, error) {
	buildCtx, buildCtxWriter := io.Pipe()
	go func() {
		err := docker.CreateDockerTarContext(ctx, buildCtxWriter, a.Workspace, a.DockerArtifact, insecureRegistries)
		if err != nil {
			buildCtxWriter.CloseWithError(errors.Wrap(err, "creating docker context"))
			return
		}
		buildCtxWriter.Close()
	}()

	return io.Copy(ioutil.Discard, buildCtx)
}
