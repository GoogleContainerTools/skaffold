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

package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

var (
	// For testing
	environ = os.Environ
)

func (b *Builder) buildCustom(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	artifact := a.CustomArtifact
	cmd := exec.Command(artifact.BuildCommand)
	cmd.Env = retrieveEnv(tag)
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

func retrieveEnv(tag string) []string {
	tags := []string{
		fmt.Sprintf("%s=%s", constants.ImageName, tag),
	}
	return append(tags, environ()...)
}
