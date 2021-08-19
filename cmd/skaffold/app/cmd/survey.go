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

var surveyID string

func NewCmdSurvey() *cobra.Command {
	return NewCmd("survey").
		WithDescription("Opens a web browser to fill out the Skaffold survey").
		WithFlags([]*Flag{
			{Value: &surveyID, Name: "id", DefValue: survey.HatsID, Usage: "Survey ID for survey command to open."},
		}).
		NoArgs(showSurvey)
}

func showSurvey(ctx context.Context, out io.Writer) error {
	s := survey.New(opts.GlobalConfig, opts.ConfigurationFile, opts.Command)
	return s.OpenSurveyForm(ctx, out, surveyID)
}
