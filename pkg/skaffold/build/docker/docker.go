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

package docker

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	pluginutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// Builder builds artifacts with Docker.
type Builder struct {
	skipTests bool
}

// NewBuilder creates a new Builder that builds artifacts with Docker.
func NewBuilder() *Builder {
	return &Builder{}
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
	m := pluginutil.GroupArtifactsByEnvironment(artifacts)
	var builds []build.Artifact

	for env, arts := range m {
		switch env.Name {
		case constants.GoogleCloudBuildExecEnv:
			build, err := b.googleCloudBuild(ctx, out, tags, arts, env)
			if err != nil {
				return nil, err
			}
			builds = append(builds, build...)
		default:
			return nil, errors.Errorf("%s is not a supported environment for builder docker", env.Name)
		}
	}
	return builds, nil
}

// googleCloudBuild sets any necessary defaults and then builds artifacts with docker in GCB
func (b *Builder) googleCloudBuild(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, env *latest.ExecutionEnvironment) ([]build.Artifact, error) {
	var g *latest.GoogleCloudBuild
	if err := util.Convert(env.Properties, &g); err != nil {
		return nil, errors.Wrap(err, "converting execution environment to googlecloudbuild struct")
	}
	defaults.SetDefaultCloudBuildDockerImage(g)
	return gcb.NewBuilder(g, b.skipTests).Build(ctx, out, tags, artifacts)
}
