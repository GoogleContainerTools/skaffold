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
	"time"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"

	sConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/timeutil"
)

const (
	Form = `Thank you for offering your feedback on Skaffold! Understanding your experiences and opinions helps us make Skaffold better for you and other users.

Skaffold will now attempt to open the survey in your default web browser. You may also manually open it using this URL:

%s

Tip: To permanently disable the survey prompt, run:
   skaffold config set --survey --global disable-prompt true`
)

var (
	// for testing
	isStdOut             = output.IsStdout
	open                 = browser.OpenURL
	updateSurveyPrompted = sConfig.UpdateGlobalSurveyPrompted
	parseConfig          = schema.ParseConfigAndUpgrade
)

type Runner struct {
	configFile     string
	skaffoldConfig string
	mode           sConfig.RunMode
}

func New(configFile string, skaffoldConfig string, mode string) *Runner {
	return &Runner{
		configFile:     configFile,
		skaffoldConfig: skaffoldConfig,
		mode:           sConfig.RunMode(mode),
	}
}

// NextSurveyID returns the survey id of the survey to be shown
// to the user. In case no survey is available, it returns empty string.
func (s *Runner) NextSurveyID() string {
	if id, ok := s.shouldDisplaySurveyPrompt(); ok {
		return id
	}
	return ""
}

func (s *Runner) shouldDisplaySurveyPrompt() (string, bool) {
	cfg, disabled := isSurveyPromptDisabled(s.configFile)
	if disabled {
		return "", !disabled
	}
	id := s.recentlyPromptedOrTaken(cfg)
	return s.recentlyPromptedOrTaken(cfg), id != ""
}

func isSurveyPromptDisabled(configfile string) (*sConfig.GlobalConfig, bool) {
	cfg, err := sConfig.ReadConfigFile(configfile)
	if err != nil {
		return nil, false
	}
	return cfg, cfg != nil && cfg.Global != nil &&
		cfg.Global.Survey != nil &&
		cfg.Global.Survey.DisablePrompt != nil &&
		*cfg.Global.Survey.DisablePrompt
}

func (s *Runner) recentlyPromptedOrTaken(cfg *sConfig.GlobalConfig) string {
	if cfg == nil || cfg.Global == nil || cfg.Global.Survey == nil {
		return s.selectSurvey(map[string]struct{}{})
	}
	if recentlyPrompted(cfg.Global.Survey) {
		return ""
	}
	return s.selectSurvey(surveysTaken(cfg.Global.Survey))
}

// recentlyPrompted returns true if the user has been recently prompted for a survey.
func recentlyPrompted(gc *sConfig.SurveyConfig) bool {
	return timeutil.LessThan(gc.LastPrompted, 10*24*time.Hour)
}

func (s *Runner) DisplaySurveyPrompt(out io.Writer, id string) error {
	if !isStdOut(out) {
		return nil
	}
	if sc, ok := getSurvey(id); ok {
		output.Green.Fprintf(out, sc.prompt())
		return updateSurveyPrompted(s.configFile)
	}
	return nil
}

func (s *Runner) OpenSurveyForm(_ context.Context, out io.Writer, id string) error {
	sc, ok := getSurvey(id)
	if !ok {
		return fmt.Errorf("invalid survey id %q - please enter one of %s", id, validKeys())
	}
	_, err := fmt.Fprintln(out, fmt.Sprintf(Form, sc.URL))
	if err != nil {
		return err
	}
	if err := open(sc.URL); err != nil {
		logrus.Debugf("could not open url %s", sc.URL)
		return err
	}

	if id == HatsID {
		return sConfig.UpdateHaTSSurveyTaken(s.configFile)
	}
	return sConfig.UpdateUserSurveyTaken(s.configFile, id)
}

func surveysTaken(sc *sConfig.SurveyConfig) map[string]struct{} {
	if sc == nil {
		return map[string]struct{}{}
	}
	taken := map[string]struct{}{}
	if timeutil.LessThan(sc.LastTaken, 90*24*time.Hour) {
		taken[HatsID] = struct{}{}
	}
	for _, cs := range sc.UserSurveys {
		if *cs.Taken {
			taken[cs.ID] = struct{}{}
		}
	}
	return taken
}

func (s *Runner) selectSurvey(takenSurveys map[string]struct{}) string {
	var candidates []config
	for _, sc := range surveys {
		if _, taken := takenSurveys[sc.id]; !taken && sc.isActive() {
			candidates = append(candidates, sc)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sortSurveys(candidates)
	cfgs, err := parseConfig(s.skaffoldConfig)
	if err != nil {
		logrus.Debugf("error parsing skaffold.yaml %s", err)
		return ""
	}
	for _, sc := range candidates {
		if sc.isRelevantFn(cfgs, s.mode) {
			return sc.id
		}
	}
	return ""
}
