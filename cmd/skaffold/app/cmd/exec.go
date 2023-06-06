/*
Copyright 2023 The Skaffold Authors

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
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
)

// NewCmdExec describes the CLI command to execute a custom action.
func NewCmdExec() *cobra.Command {
	return NewCmd("exec").
		WithDescription("Execute a custom action").
		WithExample("Execute a defined action", "exec <action-name>").
		WithExample("Execute a defined action that uses an image built from Skaffold. First, build the images", "build --file-output=build.json").
		WithExample("Then use the built artifacts", "exec <action-name> --build-artifacts=build.json").
		WithCommonFlags().
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				log.Entry(context.TODO()).Errorf("`exec` requires exactly one action to execute")
				return errors.New("`exec` requires exactly one action to execute")
			}
			return nil
		}, doExec)
}

func doExec(ctx context.Context, out io.Writer, args []string) error {
	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		buildArtifacts, err := getBuildArtifactsAndSetTagsForAction(configs, r.ApplyDefaultRepo, args[0])
		if err != nil {
			tips.PrintUseBuildAndExec(out)
			return err
		}
		return r.Exec(ctx, out, buildArtifacts, args[0])
	})
}

func getBuildArtifactsAndSetTagsForAction(configs []util.VersionedConfig, defaulterFn DefaultRepoFn, action string) ([]graph.Artifact, error) {
	imgs := getActionImgs(action, configs)

	if len(imgs) == 0 {
		return nil, nil
	}

	fromBuildArtifactsFile := joinWithArtifactsFromBuildArtifactsFile(imgs)

	// We only use the images from previous builds, read from the --build-artifacts flag.
	// `exec` itself does not perform a build, so we don't care about the configuration in the build stanza.
	buildArtifacts, err := mergeBuildArtifacts(fromBuildArtifactsFile, preBuiltImages.Artifacts(), []*latest.Artifact{})
	if err != nil {
		return nil, err
	}

	return applyDefaultRepoToArtifacts(buildArtifacts, defaulterFn)
}

func joinWithArtifactsFromBuildArtifactsFile(imgs map[string]bool) (artifacts []graph.Artifact) {
	allArtifacts := fromBuildOutputFile.BuildArtifacts()

	for _, a := range allArtifacts {
		if imgs[a.ImageName] {
			artifacts = append(artifacts, a)
		}
	}

	return
}

func getActionImgs(action string, configs []util.VersionedConfig) map[string]bool {
	var allActions []latest.Action
	imgs := map[string]bool{}

	for _, cfg := range configs {
		allActions = append(allActions, cfg.(*latest.SkaffoldConfig).CustomActions...)
	}

	var actionCfg *latest.Action = nil
	for _, a := range allActions {
		if a.Name == action {
			actionCfg = &a
			break
		}
	}

	if actionCfg != nil {
		for _, c := range actionCfg.Containers {
			imgs[c.Image] = true
		}
	}

	return imgs
}
