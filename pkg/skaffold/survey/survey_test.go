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
					LastPrompted:  "2019-01-22T00:00:00Z",
				},
			},
		},
		{
			description: "should not display prompt when last taken in less than 3 months",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     "2018-11-22T00:00:00Z",
				},
			},
		},
		{
			description: "should display prompt when last prompted is before 2 weeks",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastPrompted:  "2019-01-10T00:00:00Z",
				},
			},
			expected: true,
		},
		{
			description: "should display prompt when last taken is before than 3 months ago",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     "2017-11-10T00:00:00Z",
				},
			},
			expected: true,
		},
		{
			description: "should not display prompt when last taken is recent than 3 months ago",
			cfg: &sConfig.ContextConfig{
				Survey: &sConfig.SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     "2019-01-10T00:00:00Z",
					LastPrompted:  "2019-01-10T00:00:00Z",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&sConfig.GetConfigForCurrentKubectx, func(string) (*sConfig.ContextConfig, error) { return test.cfg, nil })
			t.Override(&current, func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2019-01-30T12:04:05Z")
				return t
			})
			t.CheckDeepEqual(test.expected, New("test").ShouldDisplaySurveyPrompt())
		})
	}
}
