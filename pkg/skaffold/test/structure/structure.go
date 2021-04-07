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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type Runner struct {
	structureTests []string
	image          string
	testWorkingDir string
	localDaemon    docker.LocalDaemon
	imagesAreLocal func(imageName string) (bool, error)
}

// New creates a new structure.Runner.
func New(cfg docker.Config, wd string, tc *latest.TestCase, imagesAreLocal func(imageName string) (bool, error)) (*Runner, error) {
	localDaemon, err := docker.NewAPIClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Runner{
		structureTests: tc.StructureTests,
		image:          tc.ImageName,
		testWorkingDir: wd,
		localDaemon:    localDaemon,
		imagesAreLocal: imagesAreLocal,
	}, nil
}

// Test is the entrypoint for running structure tests
func (cst *Runner) Test(ctx context.Context, out io.Writer, bRes []build.Artifact) error {
	event.TestInProgress()
	if err := cst.runStructureTests(ctx, out, bRes); err != nil {
		event.TestFailed(cst.image, err)
		return containerStructureTestErr(err)
	}
	event.TestComplete()
	return nil
}

func (cst *Runner) runStructureTests(ctx context.Context, out io.Writer, bRes []build.Artifact) error {
	fqn, err := cst.getImage(ctx, out, cst.image, bRes, cst.imagesAreLocal)
	if err != nil {
		return err
	}
	if fqn == "" {
		return nil
	}

	files, err := cst.TestDependencies()
	if err != nil {
		return err
	}

	logrus.Infof("Running structure tests for files %v", files)

	args := []string{"test", "-v", "warn", "--image", fqn}
	for _, f := range files {
		args = append(args, "--config", f)
	}

	cmd := exec.CommandContext(ctx, "container-structure-test", args...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = cst.env()

	if err := util.RunCmd(cmd); err != nil {
		return fmt.Errorf("error running container-structure-test command: %w", err)
	}

	return nil
}

// TestDependencies returns dependencies listed for the structure tests
func (cst *Runner) TestDependencies() ([]string, error) {
	files, err := util.ExpandPathsGlob(cst.testWorkingDir, cst.structureTests)
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

func (cst *Runner) getImage(ctx context.Context, out io.Writer, imageName string, bRes []build.Artifact,
	imagesAreLocal func(imageName string) (bool, error)) (string, error) {
	fqn, found := resolveArtifactImageTag(imageName, bRes)
	if !found {
		logrus.Debugln("Skipping tests for", imageName, "since it wasn't built")
		return "", nil
	}

	if imageIsLocal, err := imagesAreLocal(imageName); err != nil {
		return "", err
	} else if !imageIsLocal {
		// The image is remote so we have to pull it locally.
		// `container-structure-test` currently can't do it:
		// https://github.com/GoogleContainerTools/container-structure-test/issues/253.
		if err := cst.localDaemon.Pull(ctx, out, fqn); err != nil {
			return dockerPullImageErr(fqn, err)
		}
	}

	return fqn, nil
}

func resolveArtifactImageTag(imageName string, bRes []build.Artifact) (string, bool) {
	for _, res := range bRes {
		if imageName == res.ImageName {
			return res.Tag, true
		}
	}

	return "", false
}
