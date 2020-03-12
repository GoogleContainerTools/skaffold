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
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/buildimage"
)

var options buildimage.Options

// NewCmdBuildImage describes the CLI command to build one image.
func NewCmdBuildImage() *cobra.Command {
	return NewCmd("build-image").
		WithDescription("Build an image from sources").
		WithLongDescription("Build, test and tag an image from a source workspace").
		WithExample("Build and push an image", "skaffold build-image --name my-image --push=true").
		WithCommonFlags().
		WithFlags(func(f *pflag.FlagSet) {
			f.BoolVar(&options.Push, "push", false, "Push the image to a registry")
			f.StringVar(&options.Name, "name", "", "Name of the image. Defaults to the name of the current folder")
			f.StringVar(&options.Tagger, "tagger", "", "Type of the tagger. Defaults to gitCommit")
			f.StringVar(&options.Type, "type", "", "Type of the artifact. Default to docker (docker build with a Dockerfile)")

			// Docker specific flags
			f.StringVar(&options.Target, "target", "", "Dockerfile target to build")
		}).
		AtMostArgs(1, func(ctx context.Context, out io.Writer, args []string) error {
			setWorkspace(args)
			return doBuildImage(ctx, out)
		})
}

func setWorkspace(args []string) {
	options.Workspace = "."
	if len(args) == 1 {
		options.Workspace = args[0]
	}
}

func doBuildImage(ctx context.Context, out io.Writer) error {
	return buildimage.BuildImage(ctx, out, opts, options)
}
