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

package docker

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/environments/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// Builder builds artifacts with Docker.
type Builder struct {
	opts *config.SkaffoldOptions
	env  *latest.ExecutionEnvironment
}

// NewBuilder creates a new Builder that builds artifacts with Docker.
func NewBuilder() *Builder {
	return &Builder{}
}

// Init stores skaffold options and the execution environment
func (b *Builder) Init(opts *config.SkaffoldOptions, env *latest.ExecutionEnvironment) {
	b.opts = opts
	b.env = env
}

// Labels are labels specific to Docker.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "docker",
	}
}

// Build is responsible for building artifacts in their respective execution environments
// The builder plugin is also responsible for setting any necessary defaults
func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	switch b.env.Name {
	case constants.GoogleCloudBuild:
		return b.googleCloudBuild(ctx, out, tags, artifacts)
	default:
		return nil, errors.Errorf("%s is not a supported environment for builder docker", b.env.Name)
	}
}

// googleCloudBuild sets any necessary defaults and then builds artifacts with docker in GCB
func (b *Builder) googleCloudBuild(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var g *latest.GoogleCloudBuild
	if err := util.CloneThroughJSON(b.env.Properties, &g); err != nil {
		return nil, errors.Wrap(err, "converting execution environment to googleCloudBuild struct")
	}
	defaults.SetDefaultCloudBuildDockerImage(g)
	return gcb.NewBuilder(g, b.opts.SkipTests).Build(ctx, out, tags, artifacts)
}
