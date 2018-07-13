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

package build

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

func (l *LocalBuilder) buildDocker(ctx context.Context, out io.Writer, workspace string, a *v1alpha2.DockerArtifact) (string, error) {
	initialTag := util.RandomID()

	err := docker.RunBuild(ctx, out, l.api, workspace, types.ImageBuildOptions{
		Tags:       []string{initialTag},
		Dockerfile: a.DockerfilePath,
		BuildArgs:  a.BuildArgs,
		CacheFrom:  a.CacheFrom,
	})
	if err != nil {
		return "", errors.Wrap(err, "running build")
	}

	return fmt.Sprintf("%s:latest", initialTag), nil
}
