/*
Copyright 2019 The Skaffold Authors

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

package survey

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
)

const (
	Prompt = `Help improve Skaffold with our 2-minute anonymous survey: run 'skaffold survey'
`

	URL = "https://forms.gle/BMTbGQXLWSdn7vEs6"
)

var (
	Form = fmt.Sprintf(`Thank you for offering your feedback on Skaffold! Understanding your experiences and opinions helps us make Skaffold better for you and other users.

Skaffold will now attempt to open the survey in your default web browser. You may also manually open it using this link:

%s

Tip: To permanently disable the survey prompt, run:
   skaffold config set --survey --global disable-prompt true`, URL)

	// for testing
	isStdOut     = output.IsStdout
	open         = browser.OpenURL
	updateConfig = config.UpdateGlobalSurveyPrompted
)

type Runner struct {
	configFile string
}

func New(configFile string) *Runner {
	return &Runner{
		configFile: configFile,
	}
}

func (s *Runner) DisplaySurveyPrompt(out io.Writer) error {
	if isStdOut(out) {
		output.Green.Fprintf(out, Prompt)
	}
	return updateConfig(s.configFile)
}

func (s *Runner) OpenSurveyForm(_ context.Context, out io.Writer) error {
	_, err := fmt.Fprintln(out, Form)
	if err != nil {
		return err
	}
	if err := open(URL); err != nil {
		logrus.Debugf("could not open url %s", URL)
		return err
	}
	// Currently we will only update the global survey taken
	// When prompting for the survey, we need to use the same field.
	return config.UpdateGlobalSurveyTaken(s.configFile)
}
