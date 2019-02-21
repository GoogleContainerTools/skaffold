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

package kaniko

import (
	"context"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// Builder builds docker artifacts on Kubernetes, using Kaniko.
type Builder struct {
	*latest.KanikoBuild

	timeout time.Duration
}

// NewBuilder creates a new Builder that builds artifacts with Kaniko.
func NewBuilder(cfg *latest.KanikoBuild) (*Builder, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, errors.Wrap(err, "parsing timeout")
	}

	return &Builder{
		KanikoBuild: cfg,
		timeout:     timeout,
	}, nil
}

// Labels are labels specific to Kaniko builder.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "kaniko",
	}
}

// DependenciesForArtifact returns the Dockerfile dependencies for this kaniko artifact
func (b *Builder) DependenciesForArtifact(ctx context.Context, a *latest.Artifact) ([]string, error) {
	paths, err := docker.GetDependencies(ctx, a.Workspace, a.DockerArtifact)
	if err != nil {
		return nil, errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
	}
	return util.AbsolutePaths(a.Workspace, paths), nil
}
