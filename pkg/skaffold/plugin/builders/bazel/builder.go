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

package bazel

import (
	"context"
	"io"

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Builder builds artifacts with Bazel.
type Builder struct {
	opts         *config.SkaffoldOptions
	env          *latest.ExecutionEnvironment
	localDocker  docker.LocalDaemon
	localCluster bool
	pushImages   bool
	kubeContext  string
}

// NewBuilder creates a new Builder that builds artifacts with Bazel.
func NewBuilder() (*Builder, error) {
	localCluster, err := configutil.GetLocalCluster()
	if err != nil {
		return nil, errors.Wrap(err, "getting localCluster")
	}
	kubeContext, err := kubectx.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current cluster context")
	}
	localDocker, err := docker.NewAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}
	return &Builder{
		localCluster: localCluster,
		kubeContext:  kubeContext,
		localDocker:  localDocker,
	}, nil
}

// Init stores skaffold options and the execution environment
func (b *Builder) Init(opts *config.SkaffoldOptions, env *latest.ExecutionEnvironment) {
	b.opts = opts
	b.env = env
}

// Labels are labels specific to Bazel.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "bazel",
	}
}

// DependenciesForArtifact returns the dependencies for this bazel artifact
func (b *Builder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	if err := setArtifact(artifact); err != nil {
		return nil, err
	}
	if artifact.BazelArtifact == nil {
		return nil, errors.New("bazel artifact is nil")
	}
	paths, err := GetDependencies(ctx, artifact.Workspace, artifact.BazelArtifact)
	if err != nil {
		return nil, errors.Wrap(err, "getting bazel dependencies")
	}
	return util.AbsolutePaths(artifact.Workspace, paths), nil
}

// Build is responsible for building artifacts in their respective execution environments
// The builder plugin is also responsible for setting any necessary defaults
func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	switch b.env.Name {
	case constants.Local:
		return b.local(ctx, out, tags, artifacts)
	default:
		return nil, errors.Errorf("%s is not a supported environment for builder bazel", b.env.Name)
	}
}

func setArtifact(artifact *latest.Artifact) error {
	if artifact.ArtifactType.BazelArtifact != nil {
		return nil
	}
	var a *latest.BazelArtifact
	if err := yaml.UnmarshalStrict(artifact.BuilderPlugin.Contents, &a); err != nil {
		return errors.Wrap(err, "unmarshalling bazel artifact")
	}
	if a == nil {
		return errors.New("artifact is nil")
	}
	if a.BuildTarget == "" {
		return errors.Errorf("%s must have an associated build target", artifact.ImageName)
	}
	artifact.ArtifactType.BazelArtifact = a
	return nil
}
