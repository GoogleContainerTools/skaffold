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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

const (
	hatsURL = "https://forms.gle/BMTbGQXLWSdn7vEs6"
)

var (
	hats = config{
		id: constants.HaTS,
		promptText: `Help improve Skaffold with our 2-minute anonymous survey: run 'skaffold survey'
`,
		isRelevantFn: func(_ []util.VersionedConfig) bool { return true },
		link:         hatsURL,
	}
	// surveys contains all the skaffold survey information
	surveys = []config{hats}

	// for testing
	today = time.Now()
)

// config defines a survey.
type config struct {
	id           string
	promptText   string
	expiresAt    time.Time
	isRelevantFn func([]util.VersionedConfig) bool
	link         string
}

func (s config) Link() string {
	return s.link
}

func (s config) isActive() bool {
	return s.expiresAt.IsZero() || s.expiresAt.After(time.Now())
}

func (s config) prompt() string {
	if s.id == hats.id {
		return s.promptText
	}
	return fmt.Sprintf(`%s: run 'skaffold survey -id %s'
`, s.promptText, s.id)
}

func (s config) isRelevant(cfgs []util.VersionedConfig) bool {
	return s.isRelevantFn(cfgs)
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
