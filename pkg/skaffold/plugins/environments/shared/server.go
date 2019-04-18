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
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func convertPropertiesToBytes(a *latest.Artifact) error {
	//for _, a := range artifacts {
	if a.BuilderPlugin.Properties == nil {
		return nil
	}
	data, err := yaml.Marshal(a.BuilderPlugin.Properties)
	if err != nil {
		return err
	}
	a.BuilderPlugin.Contents = data
	a.BuilderPlugin.Properties = nil
	//}
	return nil
}

// EnvRPCServer is the RPC server that EnvRPC talks to, conforming to
// the requirements of net/rpc
type EnvRPCServer struct {
	Impl PluginEnv
}

func (s *EnvRPCServer) Init(runCtx *runcontext.RunContext, resp *interface{}) error {
	return s.Impl.Init(runCtx)
}

// ExecuteBuildArgs are args passed via rpc to the build plugin on DependencyForArtifact()
type ExecuteArtifactBuildArgs struct {
	Artifact    *latest.Artifact
	TagStr      string
	Description build.Description
}

func (s *EnvRPCServer) ExecuteArtifactBuild(b ExecuteArtifactBuildArgs, resp *build.Artifact) error {
	artifact, err := s.Impl.ExecuteArtifactBuild(context.Background(), os.Stdout, b.TagStr, b.Artifact, b.Description)
	if err != nil {
		return errors.Wrap(err, "building artifacts")
	}
	*resp = artifact
	return nil
}

// EnvPlugin is the implementation of the hashicorp plugin.Plugin interface
type EnvPlugin struct {
	Impl PluginEnv
}

func (p *EnvPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &EnvRPCServer{Impl: p.Impl}, nil
}

func (EnvPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &EnvRPCClient{client: c}, nil
}
