/*
Copyright 2020 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/survey"
)

func NewCmdSurvey() *cobra.Command {
	return NewCmd("survey").
		WithDescription("Show Skaffold survey url").
		NoArgs(showSurvey)
}

func showSurvey(context context.Context, out io.Writer) error {
	s := survey.New(opts.GlobalConfig)
	return s.OpenSurveyForm(context, out)
}
