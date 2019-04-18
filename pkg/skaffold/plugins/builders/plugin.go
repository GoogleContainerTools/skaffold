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

package builders

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugins/builders/shared"
	runctx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// For testing
	randomID = util.RandomFourCharacterID
)

// RegisteredBuilderPlugins initializes and returns all required plugin builders
func RegisteredBuilderPlugins(runCtx *runctx.RunContext) (shared.PluginBuilder, error) {
	// We're a host. Start by launching the plugin process.
	logrus.SetOutput(os.Stdout)

	builders := map[string]shared.PluginBuilder{}

	for _, a := range runCtx.Cfg.Build.Artifacts {
		if a.BuilderPlugin == nil {
			continue
		}
		p := a.BuilderPlugin.Name
		if _, ok := builders[p]; ok {
			continue
		}
		cmd := exec.Command(p)
		if _, ok := SkaffoldCorePluginExecutionMap[p]; ok {
			executable, err := os.Executable()
			if err != nil {
				return nil, errors.Wrap(err, "getting executable path")
			}
			cmd = exec.Command(executable, "serve-builder-plugins")
			cmd.Env = append(os.Environ(), []string{fmt.Sprintf("%s=%s", constants.SkaffoldPluginKey, constants.SkaffoldPluginValue),
				fmt.Sprintf("%s=%s", constants.SkaffoldPluginName, p)}...)
		}

		client := plugin.NewClient(&plugin.ClientConfig{
			Stderr:          os.Stderr,
			SyncStderr:      os.Stderr,
			SyncStdout:      os.Stdout,
			Managed:         true,
			HandshakeConfig: shared.Handshake,
			Plugins:         shared.PluginMap,
			Cmd:             cmd,
		})

		logrus.Debugf("Starting Build plugin with command: %+v", cmd)

		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			return nil, errors.Wrap(err, "connecting via rpc")
		}
		logrus.Debugf("build plugin started.")
		// Request the plugin
		raw, err := rpcClient.Dispense(p)
		if err != nil {
			return nil, errors.Wrap(err, "requesting rpc plugin")
		}
		pluginBuilder := raw.(shared.PluginBuilder)
		builders[p] = pluginBuilder
	}

	b := &Builder{
		Builders: builders,
	}

	logrus.Debugf("Calling Init() for all plugins.")
	if err := b.Init(runCtx); err != nil {
		plugin.CleanupClients()
		return nil, err
	}
	return b, nil
}

func InitBuilderPluginForArtifact(runCtx *runctx.RunContext, a *latest.Artifact) (shared.PluginBuilder, error) {
	// We're a host. Start by launching the plugin process.
	logrus.SetOutput(os.Stdout)
	if a.BuilderPlugin == nil {
		return nil, errors.New(fmt.Sprint("found BuilderPlugin nil for artifact %s.", a.ImageName))
	}
	p := a.BuilderPlugin.Name
	cmd := exec.Command(p)
	if _, ok := SkaffoldCorePluginExecutionMap[p]; ok {
		executable, err := os.Executable()
		if err != nil {
			return nil, errors.Wrap(err, "getting executable path")
		}
		cmd = exec.Command(executable, "serve-builder-plugins")
		cmd.Env = append(os.Environ(), []string{fmt.Sprintf("%s=%s", constants.SkaffoldPluginKey, constants.SkaffoldPluginValue),
			fmt.Sprintf("%s=%s", constants.SkaffoldPluginName, p)}...)
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		Stderr:          os.Stderr,
		SyncStderr:      os.Stderr,
		SyncStdout:      os.Stdout,
		Managed:         true,
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             cmd,
	})

	logrus.Debugf("Starting Build plugin with command: %+v", cmd)

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return nil, errors.Wrap(err, "connecting via rpc")
	}
	logrus.Debugf("build plugin started.")
	// Request the plugin
	raw, err := rpcClient.Dispense(p)
	if err != nil {
		return nil, errors.Wrap(err, "requesting rpc plugin")
	}
	pluginBuilder := raw.(shared.PluginBuilder)

	logrus.Debugf("Calling Init() for all plugins.")
	if err := pluginBuilder.Init(runCtx); err != nil {
		plugin.CleanupClients()
		return nil, err
	}
	return pluginBuilder, nil
}

type Builder struct {
	Builders map[string]shared.PluginBuilder
}

func (b *Builder) Init(runCtx *runctx.RunContext) error {
	for _, builder := range b.Builders {
		if err := builder.Init(runCtx); err != nil {
			return err
		}
	}
	return nil
}

// Labels are labels applied to deployed resources.
func (b *Builder) Labels() map[string]string {
	labels := map[string]string{}
	for _, builder := range b.Builders {
		for k, v := range builder.Labels() {
			if val, ok := labels[k]; ok {
				random := fmt.Sprintf("%s-%s", k, randomID())
				logrus.Warnf("%s=%s label exists, saving %s=%s as %s=%s to avoid overlap", k, val, k, v, random, v)
				labels[random] = v
				continue
			}
			labels[k] = v
		}
	}
	return labels
}

func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var builtArtifacts []build.Artifact
	// Group artifacts by plugin name
	m := retrieveArtifactsByPlugin(artifacts)
	// Group artifacts by builder
	for name, builder := range b.Builders {
		bArts, err := builder.Build(ctx, out, tags, m[name])
		if err != nil {
			return nil, errors.Wrapf(err, "building artifacts with builder %s", name)
		}
		builtArtifacts = append(builtArtifacts, bArts...)
	}
	return builtArtifacts, nil
}

func (b *Builder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	// Group artifacts by builder
	for name, builder := range b.Builders {
		if name != artifact.BuilderPlugin.Name {
			continue
		}
		return builder.DependenciesForArtifact(ctx, artifact)
	}
	return nil, errors.New("couldn't find plugin builder to get dependencies for artifact")
}

func (b *Builder) BuildDescription(tags tag.ImageTags, artifact *latest.Artifact) (*build.Description, error) {
	// Group artifacts by builder
	for name, builder := range b.Builders {
		if name != artifact.BuilderPlugin.Name {
			continue
		}
		return builder.BuildDescription(tags, artifact)
	}
	return nil, errors.New("couldn't find plugin builder to get dependencies for artifact")
}

func (b *Builder) Prune(ctx context.Context, out io.Writer) error {
	for name, builder := range b.Builders {
		if err := builder.Prune(ctx, out); err != nil {
			return errors.Wrapf(err, "pruning images for builder %s", name)
		}
	}
	return nil
}

func retrieveArtifactsByPlugin(artifacts []*latest.Artifact) map[string][]*latest.Artifact {
	m := map[string][]*latest.Artifact{}
	for _, a := range artifacts {
		if _, ok := m[a.BuilderPlugin.Name]; ok {
			m[a.BuilderPlugin.Name] = append(m[a.BuilderPlugin.Name], a)
			continue
		}
		m[a.BuilderPlugin.Name] = []*latest.Artifact{a}
	}
	return m
}
