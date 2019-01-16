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

package cmd

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewCmdDiagnose describes the CLI command to diagnose skaffold.
func NewCmdDiagnose(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Run a diagnostic on Skaffold",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doDiagnose(out)
		},
	}
	cmd.Flags().StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	return cmd
}

func doDiagnose(out io.Writer) error {
	_, config, err := newRunner(opts)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}

	fmt.Fprintln(out, "Skaffold version:", version.Get().GitCommit)
	fmt.Fprintln(out, "Configuration version:", config.APIVersion)
	fmt.Fprintln(out, "Number of artifacts:", len(config.Build.Artifacts))

	if err := diagnoseArtifacts(out, config.Build.Artifacts); err != nil {
		return errors.Wrap(err, "running diagnostic on artifacts")
	}

	return nil
}

func diagnoseArtifacts(out io.Writer, artifacts []*latest.Artifact) error {
	ctx := context.Background()

	for _, artifact := range artifacts {
		color.Default.Fprintf(out, "\n%s: %s\n", typeOfArtifact(artifact), artifact.ImageName)

		if artifact.DockerArtifact != nil {
			size, err := sizeOfDockerContext(ctx, artifact)
			if err != nil {
				return errors.Wrap(err, "computing the size of the Docker context")
			}

			fmt.Fprintf(out, " - Size of the context: %vbytes\n", size)
		}

		timeDeps1, deps, err := timeToListDependencies(ctx, artifact)
		if err != nil {
			return errors.Wrap(err, "listing artifact dependencies")
		}
		timeDeps2, _, err := timeToListDependencies(ctx, artifact)
		if err != nil {
			return errors.Wrap(err, "listing artifact dependencies")
		}

		fmt.Fprintln(out, " - Dependencies:", len(deps), "files")
		fmt.Fprintf(out, " - Time to list dependencies: %v (2nd time: %v)\n", timeDeps1, timeDeps2)

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

func timeToListDependencies(ctx context.Context, a *latest.Artifact) (time.Duration, []string, error) {
	start := time.Now()

	deps, err := build.DependenciesForArtifact(ctx, a)
	if err != nil {
		return 0, nil, errors.Wrap(err, "listing artifact dependencies")
	}

	return time.Since(start), deps, nil
}

func timeToComputeMTimes(deps []string) (time.Duration, error) {
	start := time.Now()

	if _, err := watch.Stat(func() ([]string, error) { return deps, nil }); err != nil {
		return 0, errors.Wrap(err, "computing modTimes")
	}

	return time.Since(start), nil
}

func sizeOfDockerContext(ctx context.Context, a *latest.Artifact) (int64, error) {
	buildCtx, buildCtxWriter := io.Pipe()
	go func() {
		err := docker.CreateDockerTarContext(ctx, buildCtxWriter, a.Workspace, a.DockerArtifact)
		if err != nil {
			buildCtxWriter.CloseWithError(errors.Wrap(err, "creating docker context"))
			return
		}
		buildCtxWriter.Close()
	}()

	return io.Copy(ioutil.Discard, buildCtx)
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
