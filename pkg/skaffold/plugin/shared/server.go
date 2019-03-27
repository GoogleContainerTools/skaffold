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
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// BuilderRPC is an implementation of an rpc client
type BuilderRPC struct {
	client *rpc.Client
}

func (b *BuilderRPC) Init(ctx *runcontext.RunContext) {
	// We don't expect a response, so we can just use interface{}
	var resp interface{}
	b.client.Call("Plugin.Init", ctx, &resp)
}

func (b *BuilderRPC) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
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

func (b *BuilderRPC) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
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

func convertPropertiesToBytes(artifacts []*latest.Artifact) error {
	for _, a := range artifacts {
		if a.BuilderPlugin.Properties == nil {
			continue
		}
		data, err := yaml.Marshal(a.BuilderPlugin.Properties)
		if err != nil {
			return err
		}
		a.BuilderPlugin.Contents = data
		a.BuilderPlugin.Properties = nil
	}
	return nil
}

// BuilderRPCServer is the RPC server that BuilderRPC talks to, conforming to
// the requirements of net/rpc
type BuilderRPCServer struct {
	Impl PluginBuilder
}

func (s *BuilderRPCServer) Init(ctx *runcontext.RunContext, resp *interface{}) error {
	s.Impl.Init(ctx)
	return nil
}

func (s *BuilderRPCServer) Labels(args interface{}, resp *map[string]string) error {
	*resp = s.Impl.Labels()
	return nil
}

func (s *BuilderRPCServer) Build(b BuildArgs, resp *[]build.Artifact) error {
	artifacts, err := s.Impl.Build(context.Background(), os.Stdout, b.ImageTags, b.Artifacts)
	if err != nil {
		return errors.Wrap(err, "building artifacts")
	}
	*resp = artifacts
	return nil
}

func (s *BuilderRPCServer) DependenciesForArtifact(d DependencyArgs, resp *[]string) error {
	dependencies, err := s.Impl.DependenciesForArtifact(context.Background(), d.Artifact)
	if err != nil {
		return errors.Wrapf(err, "getting dependencies for %s", d.Artifact.ImageName)
	}
	*resp = dependencies
	return nil
}

// DependencyArgs are args passed via rpc to the build plugin on DependencyForArtifact()
type DependencyArgs struct {
	*latest.Artifact
}

// BuildArgs are the args passed via rpc to the builder plugin on Build()
type BuildArgs struct {
	tag.ImageTags
	Artifacts []*latest.Artifact
}

// BuilderPlugin is the implementation of the hashicorp plugin.Plugin interface
type BuilderPlugin struct {
	Impl PluginBuilder
}

func (p *BuilderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &BuilderRPCServer{Impl: p.Impl}, nil
}

func (BuilderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &BuilderRPC{client: c}, nil
}
