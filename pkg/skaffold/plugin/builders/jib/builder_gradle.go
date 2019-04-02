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

package jib

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/environments/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// GradleBuilder builds artifacts with Jib.
type GradleBuilder struct {
	opts *config.SkaffoldOptions
	env  *latest.ExecutionEnvironment
	*latest.LocalBuild
	LocalDocker  docker.LocalDaemon
	LocalCluster bool
	PushImages   bool
	PluginMode   bool
	KubeContext  string
	builtImages  []string
}

// NewGradleBuilder creates a new Builder that builds artifacts with Jib.
func NewGradleBuilder() *GradleBuilder {
	return &GradleBuilder{
		PluginMode: true,
	}
}

// Init stores skaffold options and the execution environment
func (b *GradleBuilder) Init(opts *config.SkaffoldOptions, env *latest.ExecutionEnvironment) {
	if b.PluginMode {
		if err := event.SetupRPCClient(opts); err != nil {
			logrus.Warn("error establishing gRPC connection to skaffold process; events will not be handled correctly")
			logrus.Warn(err.Error())
		}
	}
	b.opts = opts
	b.env = env
}

// Labels are labels specific to Jib.
func (b *GradleBuilder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "JibGradle",
	}
}

// DependenciesForArtifact returns the dependencies for this Jib artifact
func (b *GradleBuilder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	if err := setGradleArtifact(artifact); err != nil {
		return nil, err
	}
	if artifact.JibGradleArtifact == nil {
		return nil, errors.New("Jib Gradle artifact is nil")
	}
	paths, err := jib.GetDependenciesGradle(ctx, artifact.Workspace, artifact.JibGradleArtifact)
	if err != nil {
		return nil, errors.Wrap(err, "getting Jib Gradle dependencies")
	}
	return util.AbsolutePaths(artifact.Workspace, paths), nil
}

// Build is responsible for building artifacts in their respective execution environments
// The builder plugin is also responsible for setting any necessary defaults
func (b *GradleBuilder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	switch b.env.Name {
	case constants.Local:
		return b.local(ctx, out, tags, artifacts)
	case constants.GoogleCloudBuild:
		return b.googleCloudBuild(ctx, out, tags, artifacts)
	default:
		return nil, errors.Errorf("%s is not a supported environment for builder Jib", b.env.Name)
	}
}

func (b *GradleBuilder) Prune(ctx context.Context, out io.Writer) error {
	switch b.env.Name {
	case constants.GoogleCloudBuild:
		return nil // noop
	case constants.Local:
		return b.prune(ctx, out)
	default:
		return errors.Errorf("%s is not a supported environment for builder Jib", b.env.Name)
	}
}

// googleCloudBuild sets any necessary defaults and then builds artifacts with Jib in GCB
func (b *GradleBuilder) googleCloudBuild(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var g *latest.GoogleCloudBuild
	if err := util.CloneThroughJSON(b.env.Properties, &g); err != nil {
		return nil, errors.Wrap(err, "converting execution environment to googleCloudBuild struct")
	}
	defaults.SetDefaultCloudBuildDockerImage(g)
	for _, a := range artifacts {
		if err := setGradleArtifact(a); err != nil {
			return nil, err
		}
	}
	return gcb.NewBuilder(g, b.opts.SkipTests).Build(ctx, out, tags, artifacts)
}

func setGradleArtifact(artifact *latest.Artifact) error {
	if artifact.ArtifactType.JibGradleArtifact != nil {
		return nil
	}
	var a *latest.JibGradleArtifact
	if err := yaml.UnmarshalStrict(artifact.BuilderPlugin.Contents, &a); err != nil {
		return errors.Wrap(err, "unmarshalling Jib Gradle artifact")
	}
	if a == nil {
		a = &latest.JibGradleArtifact{}
	}
	artifact.ArtifactType.JibGradleArtifact = a
	return nil
}
