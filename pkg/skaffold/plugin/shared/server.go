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
	"net/rpc"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

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

func (s *BuilderRPCServer) Init(runCtx *runcontext.RunContext, resp *interface{}) error {
	return s.Impl.Init(runCtx)
}

func (s *BuilderRPCServer) Labels(_ interface{}, resp *map[string]string) error {
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

func (s *BuilderRPCServer) Prune(args interface{}, resp *interface{}) error {
	return s.Impl.Prune(context.Background(), os.Stdout)
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
