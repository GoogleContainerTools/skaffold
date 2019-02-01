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

package plugin

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/shared"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

func NewPluginBuilder(cfg *latest.BuildConfig) (build.Builder, error) {
	// We're a host. Start by launching the plugin process.
	log.SetOutput(os.Stdout)

	plugins := map[string]struct{}{}
	for _, a := range cfg.Artifacts {
		plugins[a.Plugin.Name] = struct{}{}
	}

	builders := map[string]build.Builder{}
	for p := range plugins {
		client := plugin.NewClient(&plugin.ClientConfig{
			Stderr:          os.Stderr,
			SyncStderr:      os.Stderr,
			SyncStdout:      os.Stdout,
			Managed:         true,
			HandshakeConfig: shared.Handshake,
			Plugins:         shared.PluginMap,
			Cmd:             exec.Command(p),
		})

		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			return nil, errors.Wrap(err, "connecting via rpc")
		}

		// Request the plugin
		raw, err := rpcClient.Dispense(p)
		if err != nil {
			return nil, errors.Wrap(err, "requesting rpc plugin")
		}
		builders[p] = raw.(build.Builder)
	}

	return &Builder{
		Builders: builders,
	}, nil
}

type Builder struct {
	Builders map[string]build.Builder
}

// Labels are labels applied to deployed resources.
func (b *Builder) Labels() map[string]string {
	labels := map[string]string{}
	for _, builder := range b.Builders {
		for k, v := range builder.Labels() {
			labels[k] = v
		}
	}
	return labels
}

func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var builtArtifacts []build.Artifact
	// Group artifacts by builder
	for name, builder := range b.Builders {
		var arts []*latest.Artifact
		for _, a := range artifacts {
			if a.Plugin.Name == name {
				arts = append(arts, a)
			}
		}
		bArts, err := builder.Build(ctx, out, tagger, arts)
		if err != nil {
			return nil, errors.Wrapf(err, "building artifacts with builder %s", name)
		}
		builtArtifacts = append(builtArtifacts, bArts...)
	}
	return builtArtifacts, nil
}
