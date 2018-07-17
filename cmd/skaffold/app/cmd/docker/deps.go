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

package docker

import (
	"io"

	cmdutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/util"
	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var depsFormatFlag = flags.NewTemplateFlag("{{range .Deps}}{{.}} {{end}}\n", DepsOutput{})

func NewCmdDeps(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Returns a list of dependencies for the input dockerfile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeps(out, filename, dockerfile, context)
		},
	}
	cmd.Flags().StringVarP(&filename, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().StringVarP(&dockerfile, "dockerfile", "d", "Dockerfile", "Dockerfile path")
	cmd.Flags().StringVarP(&context, "context", "c", ".", "Dockerfile context path")
	cmd.Flags().VarP(depsFormatFlag, "output", "o", depsFormatFlag.Usage())
	return cmd
}

type DepsOutput struct {
	Deps []string
}

func runDeps(out io.Writer, filename, dockerfile, context string) error {
	config, err := cmdutil.ParseConfig(filename)
	if err != nil {
		return err
	}
	deps, err := docker.GetDependencies(getBuildArgsForDockerfile(config, dockerfile), context, dockerfile)
	if err != nil {
		return errors.Wrap(err, "getting dockerfile dependencies")
	}

	cmdOut := DepsOutput{Deps: deps}
	if err := depsFormatFlag.Template().Execute(out, cmdOut); err != nil {
		return errors.Wrap(err, "executing template")
	}
	return nil
}

func getBuildArgsForDockerfile(config *config.SkaffoldConfig, dockerfile string) map[string]*string {
	for _, artifact := range config.Build.Artifacts {
		if artifact.DockerArtifact != nil && artifact.DockerArtifact.DockerfilePath == dockerfile {
			return artifact.DockerArtifact.BuildArgs
		}
	}
	logrus.Infof("no build args found for dockerfile %s", dockerfile)
	return map[string]*string{}
}
