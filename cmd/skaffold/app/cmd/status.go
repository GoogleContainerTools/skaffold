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
	"io"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var statusTemplateFlag = flags.NewTemplateFlag(status.DefaultTemplate, status.Status{})

// NewCmdStatus describes the CLI command to retrieve information about a skaffold run
func NewCmdStatus(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Prints information about the builder/tagger/deployer associated with a skaffold run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetStatus(out, filename)
		},
	}
	AddRunDevFlags(cmd)

	cmd.Flags().StringVarP(&opts.CustomTag, "tag", "t", "", "The optional custom tag to use for images which overrides the current Tagger configuration")
	return cmd
}

func GetStatus(out io.Writer, filename string) error {
	runner, _, err := newRunner(filename)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}

	status, err := runner.Status()
	if err != nil {
		return errors.Wrap(err, "retrieving status")
	}

	if err := statusTemplateFlag.Template().Execute(out, status); err != nil {
		return errors.Wrap(err, "executing template")
	}
	return nil
}
