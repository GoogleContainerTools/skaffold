/*
Copyright 2018 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var images []string

// NewCmdDeploy describes the CLI command to deploy artifacts.
func NewCmdDeploy(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the artifacts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(out)
		},
	}
	AddRunDevFlags(cmd)
	AddRunDeployFlags(cmd)
	cmd.Flags().StringSliceVar(&images, "images", nil, "A list of images to deploy")
	return cmd
}

func runDeploy(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	r, config, err := newRunner(out, opts)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}

	var builds []build.Artifact
	for _, image := range images {
		parsed, err := docker.ParseReference(image)
		if err != nil {
			return err
		}
		builds = append(builds, build.Artifact{
			ImageName: parsed.BaseName,
			Tag:       image,
		})
	}

	if _, err := r.Deploy(ctx, out, builds); err != nil {
		return err
	}

	return r.TailLogs(ctx, out, config.Build.Artifacts, builds)
}
