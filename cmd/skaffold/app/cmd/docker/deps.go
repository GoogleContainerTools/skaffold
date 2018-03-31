/*
Copyright 2018 Google LLC

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

package docker

import (
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var depsFormatFlag = flags.NewTemplateFlag("{{range .Deps}}{{.}} {{end}}\n", DepsOutput{})

func NewCmdDeps(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Returns a list of dependencies for the input dockerfile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeps(out, filename, context)
		},
	}
	cmd.Flags().StringVarP(&filename, "filename", "f", "Dockerfile", "Dockerfile path")
	cmd.Flags().StringVarP(&context, "context", "c", ".", "Dockerfile context path")
	cmd.Flags().VarP(depsFormatFlag, "output", "o", depsFormatFlag.Usage())
	return cmd
}

type DepsOutput struct {
	Deps []string
}

func runDeps(out io.Writer, filename, context string) error {
	f, err := os.Open(filepath.Join(context, filename))
	if err != nil {
		return errors.Wrap(err, "opening dockerfile")
	}
	deps, err := docker.GetDockerfileDependencies(context, f)
	if err != nil {
		return errors.Wrap(err, "getting dockerfile dependencies")
	}
	cmdOut := DepsOutput{Deps: deps}
	if err := depsFormatFlag.Template().Execute(out, cmdOut); err != nil {
		return errors.Wrap(err, "executing template")
	}
	return nil
}
