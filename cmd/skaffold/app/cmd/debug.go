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
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	debugging "github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// for tests
var doDebug = runDebug

var filtering = false

// NewCmdDebug describes the CLI command to run a pipeline in debug mode.
// Unlike `dev`, `debug` defaults `auto-build` and `auto-deploy` to `false`.
func NewCmdDebug() *cobra.Command {
	return NewCmd("debug").
		WithDescription("[beta] Run a pipeline in debug mode").
		WithLongDescription("Similar to `dev`, but configures the pipeline for debugging.").
		WithCommonFlags().
		WithHouseKeepingMessages().
		WithFlags(func(f *pflag.FlagSet) {
			// --filter and --build-artifacts are used to support debug for Helm and are internal
			f.BoolVar(&filtering, "filter", false, `Filter manifests from stdin for debugging similar to "skaffold debug".`)
			f.MarkHidden("filter")
			f.VarP(&renderFromBuildOutputFile, "build-artifacts", "a", "File containing build result from a previous 'skaffold build --file-output'")
			f.MarkHidden("build-artifacts")
		}).
		NoArgs(func(ctx context.Context, out io.Writer) error {
			return doDebug(ctx, out)
		})
}

func runDebug(ctx context.Context, out io.Writer) error {
	if filtering {
		return doFilter(ctx, out)
	}

	opts.PortForward.ForwardPods = true
	deploy.AddManifestTransform(debugging.ApplyDebuggingTransforms)

	return doDev(ctx, out)
}

// doFilter loads the Kubernetes manifests from stdin and applies the debug transformations.
// Unlike `skaffold debug`, this filtering affects all images and not just the built artifacts.
func doFilter(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, func(r runner.Runner, cfg *latest.SkaffoldConfig) error {
		debugHelpersRegistry, err := config.GetDebugHelpersRegistry(opts.GlobalConfig)
		if err != nil {
			return err
		}
		insecureRegistries, err := getInsecureRegistries(opts, cfg)
		if err != nil {
			return err
		}

		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		manifestList := kubectl.ManifestList([][]byte{bytes})
		// if no build-artifacts are specified then all referenced images are transformed
		manifestList, err = debugging.ApplyDebuggingTransforms(manifestList, renderFromBuildOutputFile.BuildArtifacts(), deploy.Registries{
			DebugHelpersRegistry: debugHelpersRegistry,
			InsecureRegistries:   insecureRegistries,
		})
		if err != nil {
			return err
		}
		out.Write([]byte(manifestList.String()))
		return nil
	})
}

func getInsecureRegistries(opts config.SkaffoldOptions, cfg *latest.SkaffoldConfig) (map[string]bool, error) {
	cfgRegistries, err := config.GetInsecureRegistries(opts.GlobalConfig)
	if err != nil {
		return nil, err
	}
	regList := append(opts.InsecureRegistries, cfg.Build.InsecureRegistries...)
	regList = append(regList, cfgRegistries...)
	insecureRegistries := make(map[string]bool, len(regList))
	for _, r := range regList {
		insecureRegistries[r] = true
	}
	return insecureRegistries, nil
}
