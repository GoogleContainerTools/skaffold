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

package cmd

import (
	"io"
	"os"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/runner"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	filename string
)

func NewCmdRun(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "runs a pipeline file",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runSkaffold(out, filename); err != nil {
				logrus.Errorf("run: %s", err)
			}
		},
	}
	cmd.Flags().StringVarP(&filename, "filename", "f", "skaffold.yaml", "Filename of pipeline file")
	return cmd
}

func runSkaffold(out io.Writer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "opening skaffold config")
	}
	defer f.Close()

	cfg, err := config.Parse(nil, f)
	if err != nil {
		return errors.Wrap(err, "parsing skaffold config")
	}

	r, err := runner.NewForConfig(out, cfg)
	if err != nil {
		return errors.Wrap(err, "getting skaffold config")
	}

	if err := r.Run(); err != nil {
		return errors.Wrap(err, "running skaffold steps")
	}

	return nil
}
