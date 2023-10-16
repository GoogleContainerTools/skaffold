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

package structure

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type Runner struct {
	structureTests    []string
	structureTestArgs []string
	imageName         string
	imageIsLocal      bool
	workspace         string
	localDaemon       docker.LocalDaemon
}

// New creates a new structure.Runner.
func New(ctx context.Context, cfg docker.Config, tc *latest.TestCase, imageIsLocal bool) (*Runner, error) {
	localDaemon, err := docker.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Runner{
		structureTests:    tc.StructureTests,
		structureTestArgs: tc.StructureTestArgs,
		imageName:         tc.ImageName,
		workspace:         tc.Workspace,
		localDaemon:       localDaemon,
		imageIsLocal:      imageIsLocal,
	}, nil
}

// Test is the entrypoint for running structure tests
func (cst *Runner) Test(ctx context.Context, out io.Writer, imageTag string) error {
	event.TestInProgress()
	if err := cst.runStructureTests(ctx, out, imageTag); err != nil {
		event.TestFailed(cst.imageName, err)
		return containerStructureTestErr(err)
	}
	event.TestComplete()
	return nil
}

func (cst *Runner) runStructureTests(ctx context.Context, out io.Writer, imageTag string) error {
	if !cst.imageIsLocal {
		// The image is remote so we have to pull it locally.
		// `container-structure-test` currently can't do it:
		// https://github.com/GoogleContainerTools/container-structure-test/issues/253.
		if err := cst.localDaemon.Pull(ctx, out, imageTag, v1.Platform{}); err != nil {
			return dockerPullImageErr(imageTag, err)
		}
	}

	files, err := cst.TestDependencies(ctx)
	if err != nil {
		return err
	}

	log.Entry(ctx).Infof("Running structure tests for files %v", files)

	args := []string{"test", "-v", "warn", "--image", imageTag}
	for _, f := range files {
		args = append(args, "--config", f)
	}
	args = append(args, cst.structureTestArgs...)
	cmd := exec.CommandContext(ctx, "container-structure-test", args...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = cst.env()

	if err := util.RunCmd(ctx, cmd); err != nil {
		return fmt.Errorf("error running container-structure-test command: %w", err)
	}

	return nil
}

// TestDependencies returns dependencies listed for the structure tests
func (cst *Runner) TestDependencies(context.Context) ([]string, error) {
	files, err := util.ExpandPathsGlob(cst.workspace, cst.structureTests)
	if err != nil {
		return nil, expandingFilePathsErr(err)
	}

	return files, nil
}

// env returns a merged environment of the current process environment and any extra environment.
// This ensures that the correct docker environment configuration is passed to container-structure-test,
// for example when running on minikube.
func (cst *Runner) env() []string {
	extraEnv := cst.localDaemon.ExtraEnv()
	if extraEnv == nil {
		return nil
	}

	parentEnv := os.Environ()
	mergedEnv := make([]string, len(parentEnv), len(parentEnv)+len(extraEnv))
	copy(mergedEnv, parentEnv)
	return append(mergedEnv, extraEnv...)
}
