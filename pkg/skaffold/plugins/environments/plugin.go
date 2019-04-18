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

package environments

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugins/environments/shared"
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

// RegisteredEnvPlugins initializes and returns all required plugin builders
func RegisteredEnvPlugins(runCtx *runctx.RunContext) (shared.PluginEnv, error) {
	// We're a host. Start by launching the plugin process.
	logrus.SetOutput(os.Stdout)
	env := map[string]shared.PluginEnv{}

	// This is very naive way of registering all the plugins.
	// This should be replaced by a mature registration modules.
	for _, p := range []string{"googlecloudbuild"} {
		if _, ok := env[p]; ok {
			continue
		}

		executable, err := os.Executable()
		if err != nil {
			return nil, errors.Wrap(err, "getting executable path")
		}
		cmd := exec.Command(executable, "serve-env-plugin")
		cmd.Env = append(os.Environ(), []string{fmt.Sprintf("%s=%s", constants.SkaffoldEnvPluginKey, constants.SkaffoldEnvPluginValue),
			fmt.Sprintf("%s=%s", constants.SkaffoldPluginName, p)}...)

		client := plugin.NewClient(&plugin.ClientConfig{
			Stderr:          os.Stderr,
			SyncStderr:      os.Stderr,
			SyncStdout:      os.Stdout,
			Managed:         true,
			HandshakeConfig: shared.Handshake,
			Plugins:         shared.PluginMap,
			Cmd:             cmd,
		})

		logrus.Debugf("Starting Env plugin with command: %+v", cmd)

		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			return nil, errors.Wrap(err, "connecting via rpc")
		}
		logrus.Debugf("env plugin started.")
		// Request the plugin
		raw, err := rpcClient.Dispense(p)
		if err != nil {
			return nil, errors.Wrap(err, "requesting rpc plugin")
		}
		pluginEnv := raw.(shared.PluginEnv)
		env[p] = pluginEnv
	}

	b := &EnvBuilder{
		EnvBuilders: env,
	}

	logrus.Debugf("Calling Init() for all plugins.")
	if err := b.Init(runCtx); err != nil {
		plugin.CleanupClients()
		return nil, err
	}
	return b, nil
}

type EnvBuilder struct {
	EnvBuilders map[string]shared.PluginEnv
}

func (b *EnvBuilder) Init(runCtx *runctx.RunContext) error {
	for _, env := range b.EnvBuilders {
		if err := env.Init(runCtx); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteArtifactBuild
func (b *EnvBuilder) ExecuteArtifactBuild(ctx context.Context, out io.Writer, tagStr string, artifact *latest.Artifact, d build.Description) (build.Artifact, error) {
	var bArts build.Artifact
	var err error
	for name, e := range b.EnvBuilders {
		bArts, err = e.ExecuteArtifactBuild(ctx, out, tagStr, artifact, d)
		if err != nil {
			return bArts, errors.Wrapf(err, "building artifacts in execution env %s", name)
		}
	}
	return bArts, nil
}
