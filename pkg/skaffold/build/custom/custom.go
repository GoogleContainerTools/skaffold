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
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

var (
	// For testing
	environ      = os.Environ
	buildContext = retrieveBuildContext
)

// ArtifactBuilder is a builder for custom artifacts
type ArtifactBuilder struct {
	pushImages         bool
	localDocker        docker.LocalDaemon
	kubeContext        string
	insecureRegistries map[string]bool
}

func NewArtifactBuilder(pushImages bool, kubeContext string, insecureRegistries map[string]bool, localDocker docker.LocalDaemon) *ArtifactBuilder {
	return &ArtifactBuilder{
		pushImages:         pushImages,
		kubeContext:        kubeContext,
		localDocker:        localDocker,
		insecureRegistries: insecureRegistries,
	}
}

// Build builds a custom artifact and returns the digest of the built image
func (b *ArtifactBuilder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	artifact := a.CustomArtifact
	cmd := exec.Command(artifact.BuildCommand)
	env, err := b.retrieveEnv(a, tag)
	if err != nil {
		return "", errors.Wrapf(err, "retrieving env variables for %s", a.ImageName)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "building image with command %s", cmd.Args)
	}

	if b.pushImages {
		return docker.RemoteDigest(tag, b.insecureRegistries)
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *ArtifactBuilder) retrieveEnv(a *latest.Artifact, tag string) ([]string, error) {
	images := strings.Join([]string{tag}, ",")
	buildContext, err := buildContext(a.Workspace)
	if err != nil {
		return nil, errors.Wrap(err, "getting absolute path for artifact build context")
	}

	envs := []string{
		fmt.Sprintf("%s=%s", constants.Images, images),
		fmt.Sprintf("%s=%t", constants.PushImage, b.pushImages),
		fmt.Sprintf("%s=%s", constants.BuildContext, buildContext),
	}
	if b.localDocker != nil {
		envs = append(envs, b.localDocker.ExtraEnv()...)
	}
	envs = append(envs, environ()...)
	sort.Strings(envs)
	return envs, nil
}

func retrieveBuildContext(workspace string) (string, error) {
	return filepath.Abs(workspace)
}
