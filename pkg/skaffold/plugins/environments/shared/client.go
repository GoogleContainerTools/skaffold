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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"

	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
)

// EnvRPCClient is an implementation of an rpc client
type EnvRPCClient struct {
	client *rpc.Client
}

func (b *EnvRPCClient) Init(runCtx *runcontext.RunContext) error {
	// We don't expect a response, so we can just use interface{}
	var resp interface{}
	return b.client.Call("Plugin.Init", runCtx, &resp)
}

func (b *EnvRPCClient) ExecuteArtifactBuild(ctx context.Context, out io.Writer, tag string, artifact *latest.Artifact, d build.Description) (build.Artifact, error) {
	var resp build.Artifact
	if err := convertPropertiesToBytes(artifact); err != nil {
		return resp, errors.Wrapf(err, "converting properties to bytes")
	}
	args := ExecuteArtifactBuildArgs{
		Artifact:    artifact,
		Description: d,
		TagStr:      tag,
	}
	err := b.client.Call("Plugin.ExecuteArtifactBuild", args, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
