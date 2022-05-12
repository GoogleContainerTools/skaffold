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
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

var (
	showBuild bool
	offline   bool
)

// NewCmdRender describes the CLI command to build artifacts render Kubernetes manifests.
func NewCmdRender() *cobra.Command {
	return NewCmd("render").
		WithDescription("Perform all image builds, and output rendered Kubernetes manifests").
		WithExample("Hydrate Kubernetes manifests without building the images, using digest resolved from tag in remote registry ", "render --digest-source=remote").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &showBuild, Name: "loud", DefValue: false, Usage: "Show the build logs and output", IsEnum: true},
			{Value: &offline, Name: "offline", DefValue: false, Usage: `Do not connect to Kubernetes API server for manifest creation and validation. This is helpful when no Kubernetes cluster is available (e.g. GitOps model). No metadata.namespace attribute is injected in this case - the manifest content does not get changed.`, IsEnum: true},
			// This "--output" flag replaces the --render-output flag, which is deprecated.
			{Value: &opts.RenderOutput, Name: "output", Shorthand: "o", DefValue: "", Usage: "File to write rendered manifests to"},
		}).
		NoArgs(doRender)
}

func doRender(ctx context.Context, out io.Writer) error {
	// TODO(nkubala): remove this from opts in favor of a param to Build()
	opts.RenderOnly = true
	buildOut := ioutil.Discard
	if showBuild {
		buildOut = out
	}

	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		var bRes []graph.Artifact
		var err error

		if fromBuildOutputFile.String() != "" || len(preBuiltImages.GetSlice()) > 0 {
			// pass `nil` as render shouldn't build if provided --build-artifacts or --images
			bRes, err = getBuildArtifactsAndSetTags(nil, r.ApplyDefaultRepo)
			if err != nil {
				return fmt.Errorf("loading artifacts: %w", err)
			}
		} else {
			bRes, err = r.Build(ctx, buildOut, targetArtifacts(opts, configs))
			if err != nil {
				return fmt.Errorf("executing build: %w", err)
			}
		}

		manifests, errR := r.Render(ctx, out, bRes, offline, opts.RenderOutput)
		if errR != nil {
			return fmt.Errorf("rendering manifests: %w", errR)
		}
		manifest.Write(manifests.String(), opts.RenderOutput, out)
		return nil
	})
}
