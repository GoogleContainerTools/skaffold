/*
Copyright 2021 The Skaffold Authors

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
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NewCmdApply describes the CLI command to apply manifests to a cluster.
func NewCmdApply() *cobra.Command {
	return NewCmd("apply").
		WithDescription("Apply hydrated manifests to a cluster").
		WithExample("Hydrate Kubernetes pod manifest first", "render --output rendered-pod.yaml").
		WithExample("Then create resources on your cluster from that hydrated manifest", "apply rendered-pod.yaml").
		WithCommonFlags().
		WithHouseKeepingMessages().
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("`apply` requires at least one manifest argument")
			}
			return nil
		}, doApply)
}

func doApply(ctx context.Context, out io.Writer, args []string) error {
	// force set apply boolean to select default options in runner creation
	opts.Apply = true
	opts.HydratedManifests = args
	if err := validateManifests(args); err != nil {
		return err
	}
	return withRunner(ctx, out, func(r runner.Runner, configs []*latest.SkaffoldConfig) error {
		return r.Apply(ctx, out)
	})
}

func validateManifests(manifests []string) error {
	for _, m := range manifests {
		if _, err := os.Open(m); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("cannot find provided file %s", m)
			}
			return fmt.Errorf("unable to open provided file %s", m)
		}
		if !kubernetes.IsKubernetesManifest(m) {
			return fmt.Errorf("%s is not a valid Kubernetes manifest", m)
		}
	}
	return nil
}
