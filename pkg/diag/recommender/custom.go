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

package recommender

import (
	"encoding/json"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/diag/download"
	"github.com/GoogleContainerTools/skaffold/proto"
)

var (
	// testing
	downloadRules = download.HTTPDownload
)

const (
	DiagDefaultRules = "https://storage.googleapis.com/skaffold/diag/customRules/v1/rules.json"
)

type Custom struct {
	RulesPath     string
	rules         []Rule
	deployContext map[string]string
}

// NewCustom returns a custom recommender from rules file or return error
func NewCustom(path string, deployContext map[string]string) (Custom, error) {
	rules, err := loadRules(path)
	if err != nil {
		return Custom{}, err
	}
	return Custom{
		RulesPath:     path,
		rules:         rules,
		deployContext: deployContext,
	}, nil
}

func (r Custom) Make(errCode proto.StatusCode) proto.Suggestion {
	for _, rule := range r.rules {
		if rule.Matches(errCode, r.deployContext) {
			return proto.Suggestion{
				SuggestionCode: rule.SuggestionCode,
				Action:         rule.Suggestion,
			}
		}
	}
	return NilSuggestion
}

func loadRules(path string) ([]Rule, error) {
	var ruleConfig RulesConfigV1
	bytes, err := downloadRules(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &ruleConfig)
	return ruleConfig.Rules, err
}

type RulesConfigV1 struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	ErrCode              proto.StatusCode     `json:"errorCode"`
	SuggestionCode       proto.SuggestionCode `json:"suggestionCode"`
	Suggestion           string               `json:"suggestion"`
	ContextPrefixMatches map[string]string    `json:"contextPrefixMatches"`
}

func (r Rule) Matches(errCode proto.StatusCode, deployContext map[string]string) bool {
	if r.ErrCode != errCode {
		return false
	}
	for k, prefix := range r.ContextPrefixMatches {
		if value, ok := deployContext[k]; ok {
			if !strings.HasPrefix(value, prefix) {
				return false
			}
		}
	}
	return true
}
