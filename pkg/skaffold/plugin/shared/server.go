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

package shared

import (
	"context"
	"io"
	"net/rpc"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// Here is an implementation that talks over RPC
type BuilderRPC struct {
	client *rpc.Client
}

func (b *BuilderRPC) Labels() map[string]string {
	var resp map[string]string
	err := b.client.Call("Plugin.Labels", new(interface{}), &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		logrus.Error("Unable to get labels from server.")
		panic(err)
	}
	return resp
}

func (b *BuilderRPC) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, skipTests bool) ([]build.Artifact, error) {
	var resp []build.Artifact
	if err := convertPropertiesToBytes(artifacts); err != nil {
		return nil, errors.Wrapf(err, "converting properties to bytes")
	}
	args := BuilderArgs{
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
		data, err := yaml.Marshal(a.Plugin.Properties)
		if err != nil {
			return err
		}
		a.Plugin.Contents = data
		a.Plugin.Properties = nil
	}
	return nil
}

// Here is the RPC server that BuilderRPC talks to, conforming to
// the requirements of net/r
type BuilderRPCServer struct {
	Impl build.Builder
}

func (s *BuilderRPCServer) Labels(args interface{}, resp *map[string]string) error {
	*resp = s.Impl.Labels()
	return nil
}

func (s *BuilderRPCServer) Build(b BuilderArgs, resp *[]build.Artifact) error {
	artifacts, err := s.Impl.Build(context.Background(), os.Stdout, b.ImageTags, b.Artifacts)
	if err != nil {
		return errors.Wrap(err, "building artifacts")
	}
	*resp = artifacts
	return nil
}

// BuilderArgs are the args passed via rpc to the builder plugin
type BuilderArgs struct {
	tag.ImageTags
	Artifacts []*latest.Artifact
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a BuilderRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return BuilderRPC for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type BuilderPlugin struct {
	// Impl Injection
	Impl build.Builder
}

func (p *BuilderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &BuilderRPCServer{Impl: p.Impl}, nil
}

func (BuilderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &BuilderRPC{client: c}, nil
}
