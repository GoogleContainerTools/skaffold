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
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/ko/pkg/commands/options"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// addApply augments our CLI surface with apply.
func addApply(topLevel *cobra.Command) {
	po := &options.PublishOptions{}
	fo := &options.FilenameOptions{}
	so := &options.SelectorOptions{}
	bo := &options.BuildOptions{}
	apply := &cobra.Command{
		Use:   "apply -f FILENAME",
		Short: "Apply the input files with image references resolved to built/pushed image digests.",
		Long:  `This sub-command finds import path references within the provided files, builds them into Go binaries, containerizes them, publishes them, and then feeds the resulting yaml into "kubectl apply".`,
		Example: `
  # Build and publish import path references to a Docker
  # Registry as:
  #   ${KO_DOCKER_REPO}/<package name>-<hash of import path>
  # Then, feed the resulting yaml into "kubectl apply".
  # When KO_DOCKER_REPO is ko.local, it is the same as if
  # --local was passed.
  ko apply -f config/

  # Build and publish import path references to a Docker
  # Registry preserving import path names as:
  #   ${KO_DOCKER_REPO}/<import path>
  # Then, feed the resulting yaml into "kubectl apply".
  ko apply --preserve-import-paths -f config/

  # Build and publish import path references to a Docker
  # daemon as:
  #   ko.local/<import path>
  # Then, feed the resulting yaml into "kubectl apply".
  ko apply --local -f config/

  # Apply from stdin:
  cat config.yaml | ko apply -f -

  # Any flags passed after '--' are passed to 'kubectl apply' directly:
  ko apply -f config -- --namespace=foo --kubeconfig=cfg.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Validate(po, bo); err != nil {
				return fmt.Errorf("validating options: %w", err)
			}

			if !isKubectlAvailable() {
				return errors.New("error: kubectl is not available. kubectl must be installed to use ko apply")
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

			// Issue a "kubectl apply" command reading from stdin,
			// to which we will pipe the resolved files, and any
			// remaining flags passed after '--'.
			kubectlCmd := exec.CommandContext(ctx, "kubectl", append([]string{"apply", "-f", "-"}, args...)...) //nolint:gosec

			// Pass through our environment
			kubectlCmd.Env = os.Environ()
			// Pass through our std{out,err} and make our resolved buffer stdin.
			kubectlCmd.Stderr = os.Stderr
			kubectlCmd.Stdout = os.Stdout

			// Wire up kubectl stdin to resolveFilesToWriter.
			stdin, err := kubectlCmd.StdinPipe()
			if err != nil {
				return fmt.Errorf("error piping to 'kubectl apply': %w", err)
			}

			// Make sure builds are cancelled if kubectl apply fails.
			g, ctx := errgroup.WithContext(ctx)
			g.Go(func() error {
				// kubectl buffers data before starting to apply it, which
				// can lead to resources being created more slowly than desired.
				// In the case of --watch, it can lead to resources not being
				// applied at all until enough iteration has occurred.  To work
				// around this, we prime the stream with a bunch of empty objects
				// which kubectl will discard.
				// See https://github.com/google/go-containerregistry/pull/348
				for i := 0; i < 1000; i++ {
					stdin.Write([]byte("---\n"))
				}
				// Once primed kick things off.
				return ResolveFilesToWriter(ctx, builder, publisher, fo, so, stdin)
			})

			g.Go(func() error {
				// Run it.
				if err := kubectlCmd.Run(); err != nil {
					return fmt.Errorf("error executing 'kubectl apply': %w", err)
				}
				return nil
			})

			return g.Wait()
		},
	}
	options.AddPublishArg(apply, po)
	options.AddFileArg(apply, fo)
	options.AddSelectorArg(apply, so)
	options.AddBuildOptions(apply, bo)

	topLevel.AddCommand(apply)
}
