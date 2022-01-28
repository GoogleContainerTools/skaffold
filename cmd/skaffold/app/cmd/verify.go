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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// NewCmdVerify describes the CLI command to verify artifacts.
func NewCmdVerify() *cobra.Command {
	return NewCmd("verify").
		WithDescription("Run verification tests against skaffold deployments").
		WithExample("Deploy with skaffold and then verify deployments", "deploy -q | skaffold verify").
		WithCommonFlags().
		WithHouseKeepingMessages().
		NoArgs(doVerify)
}

func doVerify(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		var artifacts []*latestV1.Artifact
		for _, cfg := range configs {
			artifacts = append(artifacts, cfg.(*latestV1.SkaffoldConfig).Build.Artifacts...)
		}
		buildArtifacts, err := getBuildArtifactsAndSetTags(artifacts, r.ApplyDefaultRepo)
		if err != nil {
			tips.PrintUseRunVsDeploy(out)
			return err
		}

		return r.VerifyAndLog(ctx, out, buildArtifacts)
	})
}
