/*
Copyright 2018 Google LLC

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

package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
)

func (l *LocalBuilder) buildBazel(ctx context.Context, out io.Writer, a *v1alpha2.Artifact) (string, error) {
	cmd := exec.Command("bazel", "build", a.BazelArtifact.BuildTarget)
	cmd.Dir = a.Workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	//TODO(r2d4): strip off leading //:, bad
	tarPath := strings.TrimPrefix(a.BazelArtifact.BuildTarget, "//:")
	//TODO(r2d4): strip off trailing .tar, even worse
	imageTag := strings.TrimSuffix(tarPath, ".tar")

	imageTar, err := os.Open(filepath.Join(a.Workspace, "bazel-bin", tarPath))
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()

	resp, err := l.api.ImageLoad(ctx, imageTar, false)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}
	defer resp.Body.Close()

	err = docker.StreamDockerMessages(out, resp.Body, nil)
	if err != nil {
		return "", errors.Wrap(err, "reading from image load response")
	}

	return fmt.Sprintf("bazel:%s", imageTag), nil
}
