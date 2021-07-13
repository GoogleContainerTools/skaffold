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

package survey

import (
	"fmt"
	"sort"
	"time"

	sConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

const (
	HatsID  = "hats"
	hatsURL = "https://forms.gle/BMTbGQXLWSdn7vEs6"
)

var (
	hats = config{
		id:         HatsID,
		promptText: "Help improve Skaffold with our 2-minute anonymous survey",
		isRelevantFn: func([]util.VersionedConfig, sConfig.RunMode) bool {
			return true
		},
		URL: hatsURL,
	}
	// surveys contains all the skaffold survey information
	surveys = []config{
		hats,
		{
			id:       "helm",
			startsAt: time.Date(2021, time.July, 15, 0, 0, 0, 0, time.UTC),
			expiresAt: time.Date(2021, time.August,
				14, 00, 00, 00, 0, time.UTC),
			promptText: "Help improve Skaffold's Helm support by taking our 2-minute anonymous survey!",
			isRelevantFn: func(cfgs []util.VersionedConfig, _ sConfig.RunMode) bool {
				for _, cfg := range cfgs {
					v1Cfg, ok := cfg.(*latestV1.SkaffoldConfig)
					if !ok {
						return false
					}
					if h := v1Cfg.Deploy.HelmDeploy; h != nil {
						return true
					}
				}
				return false
			},
			URL: "https://forms.gle/cLQg8sGD71JnPSZf6",
		},
	}
)

// config defines a survey.
type config struct {
	id string
	// promptText is shown to the user and should be formatted so each line should fit in < 80 characters.
	// For example: `As a Helm user, we are requesting your feedback on a proposed change to Skaffold's integration with Helm.`
	promptText string
	// startsAt mentions the date after the users survey should be prompted. This will ensure, Skaffold team can finalize the survey
	// even after release date.
	startsAt time.Time
	// expiresAt places a time limit of the user survey. As users are only prompted every two weeks
	// by design, this time limit should be at least 4 weeks after the upcoming release date to account
	// for release propagation lag to Cloud SDK and Cloud Shell.
	expiresAt    time.Time
	isRelevantFn func([]util.VersionedConfig, sConfig.RunMode) bool
	URL          string
}

func (s config) isActive() bool {
	return s.expiresAt.IsZero() ||
		(s.startsAt.Before(time.Now()) && s.expiresAt.After(time.Now()))
}

func (s config) prompt() string {
	if s.id == hats.id {
		return fmt.Sprintf(`%s: run 'skaffold survey'
`, s.promptText)
	}
	return fmt.Sprintf(`%s: run 'skaffold survey --id %s'
`, s.promptText, s.id)
}

func (s config) isRelevant(cfgs []util.VersionedConfig, cmd sConfig.RunMode) bool {
	return s.isRelevantFn(cfgs, cmd)
}

func (s config) isValid() bool {
	if s.id == HatsID {
		return true
	}
	today := s.startsAt
	if today.IsZero() {
		today = time.Now()
	}
	return s.expiresAt.Sub(today) < 60*24*time.Hour
}

func getSurvey(id string) (config, bool) {
	for _, s := range surveys {
		if s.id == id {
			return s, true
		}
	}
	return config{}, false
}

func validKeys() []string {
	keys := make([]string, 0, len(surveys))
	for _, s := range surveys {
		keys = append(keys, s.id)
	}
	return keys
}

func init() {
	for _, s := range surveys {
		if !s.isValid() {
			panic(fmt.Errorf("survey %q is valid for more than a 60 days - user surveys must be valid for 60 days or less", s.id))
		}
	}
}

// sortSurveys sorts a slice of config based on expiry time in
// the ascending order.
// Survey that don't have expiry set are returned last.
func sortSurveys(s []config) []config {
	sort.Slice(s, func(i, j int) bool {
		if s[i].expiresAt.IsZero() { // i > j
			return false
		}
		if s[j].expiresAt.IsZero() { // i < j
			return true
		}
		return s[i].expiresAt.Before(s[j].expiresAt) // i < j
	})
	return s
}
