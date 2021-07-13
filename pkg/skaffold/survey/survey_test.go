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
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	sConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	schemaUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDisplaySurveyForm(t *testing.T) {
	tests := []struct {
		description        string
		mockSurveyPrompted func(_ string) error
		expected           string
		mockStdOut         bool
	}{
		{
			description:        "std out",
			mockStdOut:         true,
			mockSurveyPrompted: func(_ string) error { return nil },
			expected: `Help improve Skaffold with our 2-minute anonymous survey: run 'skaffold survey'
`,
		},
		{
			description: "not std out",
			mockSurveyPrompted: func(_ string) error {
				return fmt.Errorf("not expected to call")
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			mock := func(io.Writer) bool { return test.mockStdOut }
			t.Override(&isStdOut, mock)
			mockOpen := func(string) error { return nil }
			t.Override(&open, mockOpen)
			t.Override(&updateSurveyPrompted, test.mockSurveyPrompted)
			var buf bytes.Buffer
			err := New("test", "skaffold.yaml", "dev").DisplaySurveyPrompt(&buf, HatsID)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}

func TestShouldDisplayPrompt(t *testing.T) {
	tenDaysAgo := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)
	fiveDaysAgo := time.Now().AddDate(0, 0, -5).Format(time.RFC3339)
	// less than 90 days ago
	twoMonthsAgo := time.Now().AddDate(0, -2, -5).Format(time.RFC3339)
	// at least 90 days ago
	threeMonthsAgo := time.Now().AddDate(0, -3, -5).Format(time.RFC3339)

	tests := []struct {
		description string
		cfg         *sConfig.GlobalConfig
		expected    bool
	}{
		{
			description: "should not display prompt when prompt is disabled",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{
					Survey: &sConfig.SurveyConfig{DisablePrompt: util.BoolPtr(true)},
				}},
		},
		{
			description: "should not display prompt when last prompted is less than 2 weeks",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{
					Survey: &sConfig.SurveyConfig{
						DisablePrompt: util.BoolPtr(false),
						LastPrompted:  fiveDaysAgo,
					}},
			},
		},
		{
			description: "should not display prompt when last taken in less than 3 months",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{
					Survey: &sConfig.SurveyConfig{
						DisablePrompt: util.BoolPtr(false),
						LastTaken:     twoMonthsAgo,
					}},
			},
		},
		{
			description: "should display prompt when last prompted is before 2 weeks",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{
					Survey: &sConfig.SurveyConfig{
						DisablePrompt: util.BoolPtr(false),
						LastPrompted:  tenDaysAgo,
					}},
			},
			expected: true,
		},
		{
			description: "should display prompt when last taken is before than 3 months ago",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{
					Survey: &sConfig.SurveyConfig{
						DisablePrompt: util.BoolPtr(false),
						LastTaken:     threeMonthsAgo,
					}},
			},
			expected: true,
		},
		{
			description: "should not display prompt when last taken is recent than 3 months ago",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{
					Survey: &sConfig.SurveyConfig{
						DisablePrompt: util.BoolPtr(false),
						LastTaken:     twoMonthsAgo,
						LastPrompted:  twoMonthsAgo,
					}},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&surveys, []config{hats})
			t.Override(&sConfig.ReadConfigFile, func(string) (*sConfig.GlobalConfig, error) { return test.cfg, nil })
			t.Override(&parseConfig, func(string) ([]schemaUtil.VersionedConfig, error) {
				return []schemaUtil.VersionedConfig{mockVersionedConfig{version: "test"}}, nil
			})
			_, actual := New("test", "yaml", "dev").shouldDisplaySurveyPrompt()
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestIsSurveyPromptDisabled(t *testing.T) {
	tests := []struct {
		description string
		cfg         *sConfig.GlobalConfig
		readErr     error
		expected    bool
	}{
		{
			description: "config disable-prompt is nil returns false",
			cfg:         &sConfig.GlobalConfig{},
		},
		{
			description: "config disable-prompt is true",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{DisablePrompt: util.BoolPtr(true)}},
			},
			expected: true,
		},
		{
			description: "config disable-prompt is false",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{DisablePrompt: util.BoolPtr(false)}},
			},
		},
		{
			description: "disable prompt is nil",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{}},
			},
		},
		{
			description: "config is nil",
			cfg:         nil,
		},
		{
			description: "config has err",
			cfg:         nil,
			readErr:     fmt.Errorf("error while reading"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&sConfig.ReadConfigFile, func(string) (*sConfig.GlobalConfig, error) { return test.cfg, test.readErr })
			_, actual := isSurveyPromptDisabled("dummyconfig")
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestRecentlyPromptedOrTaken(t *testing.T) {
	// less than 90 days ago
	twoMonthsAgo := time.Now().AddDate(0, -2, -5).Format(time.RFC3339)
	// at least 90 days ago
	threeMonthsAgo := time.Now().AddDate(0, -3, -5).Format(time.RFC3339)
	future := time.Now().AddDate(1, 0, 0)
	tests := []struct {
		description string
		cfg         *sConfig.GlobalConfig
		input       []config
		expected    string
	}{
		{
			description: "nil test - do not remove",
			cfg:         nil,
			input:       []config{hats},
			expected:    HatsID,
		},

		// Current world when no user surveys are configured.
		{
			description: "no user surveys - hats not taken",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{}}},
			input:    []config{hats},
			expected: HatsID,
		},
		{
			description: "no user surveys - hats taken more than 3 months",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{LastTaken: threeMonthsAgo}}},
			input:    []config{hats},
			expected: HatsID,
		},
		{
			description: "no user surveys - hats taken less than 3 months",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{LastTaken: twoMonthsAgo}}},
			input:    []config{hats},
			expected: "",
		},
		// User survey configured and are relevant
		{
			description: "user surveys, hats not taken, relevant survey",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{}}},
			input: []config{hats, {id: "user", expiresAt: future,
				isRelevantFn: func(_ []schemaUtil.VersionedConfig, _ sConfig.RunMode) bool {
					return true
				}},
			},
			expected: "user",
		},
		{
			description: "user survey taken, hats taken",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{LastTaken: twoMonthsAgo,
					UserSurveys: []*sConfig.UserSurvey{
						{ID: "user", Taken: util.BoolPtr(true)},
					}}}},
			input: []config{hats, {id: "user", expiresAt: future,
				isRelevantFn: func(_ []schemaUtil.VersionedConfig, _ sConfig.RunMode) bool {
					return true
				}},
			},
			expected: "",
		},
		{
			description: "user survey taken, hats not taken",
			cfg: &sConfig.GlobalConfig{
				Global: &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{
					UserSurveys: []*sConfig.UserSurvey{
						{ID: "user", Taken: util.BoolPtr(true)},
					}}}},
			input: []config{hats, {id: "user", expiresAt: future,
				isRelevantFn: func(_ []schemaUtil.VersionedConfig, _ sConfig.RunMode) bool {
					return true
				}},
			},
			expected: HatsID,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&surveys, test.input)
			t.Override(&parseConfig, func(string) ([]schemaUtil.VersionedConfig, error) {
				return []schemaUtil.VersionedConfig{mockVersionedConfig{version: "test"}}, nil
			})
			actual := New("dummy", "yaml", "cmd").recentlyPromptedOrTaken(test.cfg)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
