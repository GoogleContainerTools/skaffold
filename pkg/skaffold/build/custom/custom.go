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

package custom

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// For testing
	buildContext = retrieveBuildContext
)

// ArtifactBuilder is a builder for custom artifacts
type ArtifactBuilder struct {
	pushImages    bool
	additionalEnv []string
}

// NewArtifactBuilder returns a new custom artifact builder
func NewArtifactBuilder(pushImages bool, additionalEnv []string) *ArtifactBuilder {
	return &ArtifactBuilder{
		pushImages:    pushImages,
		additionalEnv: additionalEnv,
	}
}

// Build builds a custom artifact
// It returns true if the image is expected to exist remotely, or false if it is expected to exist locally
func (b *ArtifactBuilder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) error {
	cmd, err := b.retrieveCmd(out, a, tag)
	if err != nil {
		return errors.Wrap(err, "retrieving cmd")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting cmd")
	}

	return b.handleGracefulTermination(ctx, cmd)
}

func (b *ArtifactBuilder) handleGracefulTermination(ctx context.Context, cmd *exec.Cmd) error {
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			// On windows we can't send specific signals to processes, so we kill the process immediately
			if runtime.GOOS == "windows" {
				cmd.Process.Kill()
				return
			}

			logrus.Debugf("Sending SIGINT to process %v\n", cmd.Process.Pid)
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				// kill process on error
				cmd.Process.Kill()
			}

			// wait 2 seconds or wait for the process to complete
			select {
			case <-time.After(2 * time.Second):
				logrus.Debugf("Killing process %v\n", cmd.Process.Pid)
				// forcefully kill process after 2 seconds grace period
				cmd.Process.Kill()
			case <-done:
				return
			}
		case <-done:
			return
		}
	}()

	return cmd.Wait()
}

func (b *ArtifactBuilder) retrieveCmd(out io.Writer, a *latest.Artifact, tag string) (*exec.Cmd, error) {
	artifact := a.CustomArtifact
	split := strings.Split(artifact.BuildCommand, " ")

	cmd := exec.Command(split[0], split[1:]...)
	cmd.Stdout = out
	cmd.Stderr = out

	env, err := b.retrieveEnv(a, tag)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving env variables for %s", a.ImageName)
	}
	cmd.Env = env

	dir, err := buildContext(a.Workspace)
	if err != nil {
		return nil, errors.Wrap(err, "getting context for artifact")
	}
	cmd.Dir = dir

	return cmd, nil
}

func (b *ArtifactBuilder) retrieveEnv(a *latest.Artifact, tag string) ([]string, error) {
	images := strings.Join([]string{tag}, " ")
	buildContext, err := buildContext(a.Workspace)
	if err != nil {
		return nil, errors.Wrap(err, "getting absolute path for artifact build context")
	}

	envs := []string{
		fmt.Sprintf("%s=%s", constants.Images, images),
		fmt.Sprintf("%s=%t", constants.PushImage, b.pushImages),
		fmt.Sprintf("%s=%s", constants.BuildContext, buildContext),
	}
	envs = append(envs, b.additionalEnv...)
	envs = append(envs, util.OSEnviron()...)
	return envs, nil
}

func retrieveBuildContext(workspace string) (string, error) {
	return filepath.Abs(workspace)
}
