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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDisplaySurveyForm(t *testing.T) {
	tests := []struct {
		description string
		mockStdOut  bool
		expected    string
	}{
		{
			description: "std out",
			mockStdOut:  true,
			expected: `Help improve Skaffold with our 2-minute anonymous survey: run 'skaffold survey'
`,
		},
		{
			description: "not std out",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			mock := func(io.Writer) bool { return test.mockStdOut }
			t.Override(&isStdOut, mock)
			mockOpen := func(string) error { return nil }
			t.Override(&open, mockOpen)
			t.Override(&updateConfig, func(_ string) error { return nil })
			var buf bytes.Buffer
			New("test").DisplaySurveyPrompt(&buf)
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
		cfg         *sConfig.ContextConfig
		expected    bool
	}{
		{
			description: "should not display prompt when prompt is disabled",
			cfg:         &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{DisablePrompt: util.BoolPtr(true)}},
		},
		{
			description: "should not display prompt when last prompted is less than 2 weeks",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastPrompted:  fiveDaysAgo,
				},
			},
		},
		{
			description: "should not display prompt when last taken in less than 3 months",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     twoMonthsAgo,
				},
			},
		},
		{
			description: "should display prompt when last prompted is before 2 weeks",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastPrompted:  tenDaysAgo,
				},
			},
			expected: true,
		},
		{
			description: "should display prompt when last taken is before than 3 months ago",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     threeMonthsAgo,
				},
			},
			expected: true,
		},
		{
			description: "should not display prompt when last taken is recent than 3 months ago",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     twoMonthsAgo,
					LastPrompted:  twoMonthsAgo,
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&sConfig.GetConfigForCurrentKubectx, func(string) (*sConfig.ContextConfig, error) { return test.cfg, nil })
			t.CheckDeepEqual(test.expected, New("test").ShouldDisplaySurveyPrompt())
		})
	}
}

func TestIsSurveyPromptDisabled(t *testing.T) {
	tests := []struct {
		description string
		cfg         *sConfig.ContextConfig
		readErr     error
		expected    bool
	}{
		{
			description: "config disable-prompt is nil returns false",
			cfg:         &sConfig.ContextConfig{},
		},
		{
			description: "config disable-prompt is true",
			cfg:         &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{DisablePrompt: util.BoolPtr(true)}},
			expected:    true,
		},
		{
			description: "config disable-prompt is false",
			cfg:         &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{DisablePrompt: util.BoolPtr(false)}},
		},
		{
			description: "disable prompt is nil",
			cfg:         &sConfig.ContextConfig{Survey: &sConfig.SurveyConfig{}},
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
			t.Override(&sConfig.GetConfigForCurrentKubectx, func(string) (*sConfig.ContextConfig, error) { return test.cfg, test.readErr })
			_, actual := isSurveyPromptDisabled("dummyconfig")
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
