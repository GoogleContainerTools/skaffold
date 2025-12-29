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
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
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
		var buildArtifacts []graph.Artifact
		var err error

		// If pre-built artifacts are provided via --build-artifacts flag, use them; otherwise build
		if fromBuildOutputFile.String() != "" {
			buildArtifacts, err = getBuildArtifactsAndSetTagsForVerify(configs, r.ApplyDefaultRepo)
			if err != nil {
				tips.PrintUseRunVsDeploy(out)
				return err
			}
		} else {
			// Build artifacts that are referenced in verify test cases
			buildArtifacts, err = r.Build(ctx, out, targetArtifactsForVerify(configs))
			if err != nil {
				return fmt.Errorf("failed to build: %w", err)
			}
		}

		defer func() {
			if err := r.Cleanup(context.Background(), out, false, manifest.NewManifestListByConfig(), opts.Command); err != nil {
				log.Entry(ctx).Warn("verifier cleanup:", err)
			}
		}()

		return r.VerifyAndLog(ctx, out, buildArtifacts)
	})
}

func getBuildArtifactsAndSetTagsForVerify(configs []util.VersionedConfig, defaulterFn DefaultRepoFn) ([]graph.Artifact, error) {
	verifyImgs := getVerifyImgs(configs)

	allImgs := joinWithArtifactsFromBuildArtifactsFile(verifyImgs)

	buildArtifacts, err := mergeBuildArtifacts(allImgs, preBuiltImages.Artifacts(), []*latest.Artifact{})
	if err != nil {
		return nil, err
	}

	return applyDefaultRepoToArtifacts(buildArtifacts, defaulterFn)
}

func getVerifyImgs(configs []util.VersionedConfig) map[string]bool {
	imgs := make(map[string]bool)
	for _, cfg := range configs {
		for _, vtc := range cfg.(*latest.SkaffoldConfig).Verify {
			imgs[vtc.Container.Image] = true
		}
	}

	return imgs
}

// targetArtifactsForVerify returns the build artifacts that are referenced by verify test cases.
// Only artifacts whose image names are used in verify containers will be built.
func targetArtifactsForVerify(configs []util.VersionedConfig) []*latest.Artifact {
	verifyImgs := getVerifyImgs(configs)

	var artifacts []*latest.Artifact
	for _, cfg := range configs {
		for _, artifact := range cfg.(*latest.SkaffoldConfig).Build.Artifacts {
			// Only include artifacts that are referenced in verify test cases
			if verifyImgs[artifact.ImageName] {
				artifacts = append(artifacts, artifact)
			}
		}
	}
	return artifacts
}
