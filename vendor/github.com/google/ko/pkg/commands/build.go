// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"

	"github.com/google/ko/pkg/commands/options"
	"github.com/spf13/cobra"
)

// addBuild augments our CLI surface with build.
func addBuild(topLevel *cobra.Command) {
	po := &options.PublishOptions{}
	bo := &options.BuildOptions{}

	build := &cobra.Command{
		Use:     "build IMPORTPATH...",
		Short:   "Build and publish container images from the given importpaths.",
		Long:    `This sub-command builds the provided import paths into Go binaries, containerizes them, and publishes them.`,
		Aliases: []string{"publish"},
		Example: `
  # Build and publish import path references to a Docker Registry as:
  #   ${KO_DOCKER_REPO}/<package name>-<hash of import path>
  # When KO_DOCKER_REPO is ko.local, it is the same as if --local and
  # --preserve-import-paths were passed.
  # If the import path is not provided, the current working directory is the
  # default.
  ko build github.com/foo/bar/cmd/baz github.com/foo/bar/cmd/blah

  # Build and publish a relative import path as:
  #   ${KO_DOCKER_REPO}/<package name>-<hash of import path>
  # When KO_DOCKER_REPO is ko.local, it is the same as if --local and
  # --preserve-import-paths were passed.
  ko build ./cmd/blah

  # Build and publish a relative import path as:
  #   ${KO_DOCKER_REPO}/<import path>
  # When KO_DOCKER_REPO is ko.local, it is the same as if --local was passed.
  ko build --preserve-import-paths ./cmd/blah

  # Build and publish import path references to a Docker daemon as:
  #   ko.local/<import path>
  # This always preserves import paths.
  ko build --local github.com/foo/bar/cmd/baz github.com/foo/bar/cmd/blah`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Validate(po, bo); err != nil {
				return fmt.Errorf("validating options: %w", err)
			}

			if len(args) == 0 {
				// Build the current directory by default.
				args = []string{"."}
			}

			ctx := cmd.Context()

			bo.InsecureRegistry = po.InsecureRegistry
			builder, err := makeBuilder(ctx, bo)
			if err != nil {
				return fmt.Errorf("error creating builder: %w", err)
			}
			publisher, err := makePublisher(po)
			if err != nil {
				return fmt.Errorf("error creating publisher: %w", err)
			}
			defer publisher.Close()
			images, err := publishImages(ctx, args, publisher, builder)
			if err != nil {
				return fmt.Errorf("failed to publish images: %w", err)
			}
			for _, img := range images {
				fmt.Println(img)
			}
			return nil
		},
	}
	options.AddPublishArg(build, po)
	options.AddBuildOptions(build, bo)
	topLevel.AddCommand(build)
}
