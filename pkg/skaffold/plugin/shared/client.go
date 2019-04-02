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

package shared

import (
	"context"
	"io"
	"net/rpc"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// BuilderRPC is an implementation of an rpc client
type BuilderRPC struct {
	client *rpc.Client
}

func (b *BuilderRPC) Init(opts *config.SkaffoldOptions, env *latest.ExecutionEnvironment) {
	// We don't expect a response, so we can just use interface{}
	var resp interface{}
	args := InitArgs{
		Opts: opts,
		Env:  env,
	}
	b.client.Call("Plugin.Init", args, &resp)
}

func (b *BuilderRPC) DependenciesForArtifact(_ context.Context, artifact *latest.Artifact) ([]string, error) {
	var resp []string
	if err := convertPropertiesToBytes([]*latest.Artifact{artifact}); err != nil {
		return nil, errors.Wrapf(err, "converting properties to bytes")
	}
	args := DependencyArgs{artifact}
	err := b.client.Call("Plugin.DependenciesForArtifact", args, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *BuilderRPC) Labels() map[string]string {
	var resp map[string]string
	err := b.client.Call("Plugin.Labels", new(interface{}), &resp)
	if err != nil {
		// Can't return error, so log it instead
		logrus.Errorf("Unable to get labels from server: %v", err)
	}
	return resp
}

func (b *BuilderRPC) Build(_ context.Context, _ io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var resp []build.Artifact
	if err := convertPropertiesToBytes(artifacts); err != nil {
		return nil, errors.Wrapf(err, "converting properties to bytes")
	}
	args := BuildArgs{
		ImageTags: tags,
		Artifacts: artifacts,
	}
	err := b.client.Call("Plugin.Build", args, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *BuilderRPC) Prune(ctx context.Context, out io.Writer) error {
	var resp interface{}
	if err := b.client.Call("Plugin.Prune", new(interface{}), &resp); err != nil {
		return err
	}
	return nil
}
