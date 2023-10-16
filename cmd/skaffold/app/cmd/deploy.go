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

package cmd

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
)

// NewCmdDeploy describes the CLI command to deploy artifacts.
func NewCmdDeploy() *cobra.Command {
	return NewCmd("deploy").
		WithDescription("Deploy pre-built artifacts").
		WithExample("Build the artifacts and collect the tags into a file", "build --file-output=tags.json").
		WithExample("Deploy those tags", "deploy --build-artifacts=tags.json").
		WithExample("Build the artifacts and then deploy them", "build -q | skaffold deploy --build-artifacts -").
		WithCommonFlags().
		WithHouseKeepingMessages().
		NoArgs(doDeploy)
}

func doDeploy(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		var artifacts []*latest.Artifact
		for _, cfg := range configs {
			artifacts = append(artifacts, cfg.(*latest.SkaffoldConfig).Build.Artifacts...)
		}
		buildArtifacts, err := getBuildArtifactsAndSetTags(artifacts, r.ApplyDefaultRepo)
		if err != nil {
			tips.PrintUseRunVsDeploy(out)
			return err
		}
		// Render
		manifests, errR := r.Render(ctx, out, buildArtifacts, false)
		if errR != nil {
			return errR
		}
		return r.DeployAndLog(ctx, out, buildArtifacts, manifests)
	})
}
