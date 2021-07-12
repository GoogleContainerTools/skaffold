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
	"testing"
	"time"

	sConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSurveyPrompt(t *testing.T) {
	tests := []struct {
		description string
		s           config
		expected    string
	}{
		{
			description: "hats survey",
			s:           hats,
			expected: `Help improve Skaffold with our 2-minute anonymous survey: run 'skaffold survey'
`,
		},
		{
			description: "not hats survey",
			s: config{
				id:         "foo",
				promptText: "Looks like you are using foo feature. Help improve Skaffold foo feature and take this survey",
				expiresAt:  time.Date(2021, time.August, 14, 00, 00, 00, 0, time.UTC),
			},
			expected: `Looks like you are using foo feature. Help improve Skaffold foo feature and take this survey: run 'skaffold survey -id foo'
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.s.prompt(), test.expected)
		})
	}
}

func TestSurveyActive(t *testing.T) {
	tests := []struct {
		description string
		s           config
		expected    bool
	}{
		{
			description: "no expiry",
			s:           hats,
			expected:    true,
		},
		{
			description: "expiry in past",
			s: config{
				id:        "expired",
				expiresAt: time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			description: "expiry in future",
			s: config{
				id:        "active",
				expiresAt: time.Now().AddDate(1, 0, 0),
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.s.isActive(), test.expected)
		})
	}
}

func TestSurveyRelevant(t *testing.T) {
	testMock := mockVersionedConfig{version: "test"}
	prodMock := mockVersionedConfig{version: "prod"}

	tests := []struct {
		description string
		s           config
		cfgs        []util.VersionedConfig
		expected    bool
	}{
		{
			description: "hats is always relevant",
			s:           hats,
			expected:    true,
		},
		{
			description: "relevant based on input configs",
			s: config{
				id: "foo",
				isRelevantFn: func(cfgs []util.VersionedConfig, _ sConfig.RunMode) bool {
					return len(cfgs) > 1
				},
			},
			cfgs:     []util.VersionedConfig{testMock, prodMock},
			expected: true,
		},
		{
			description: "not relevant based on config",
			s: config{
				id: "foo",
				isRelevantFn: func(cfgs []util.VersionedConfig, _ sConfig.RunMode) bool {
					return len(cfgs) > 1
				},
			},
			cfgs: []util.VersionedConfig{testMock},
		},
		{
			description: "contains a config with test version",
			s: config{
				id: "version-value-test",
				isRelevantFn: func(cfgs []util.VersionedConfig, _ sConfig.RunMode) bool {
					for _, cfg := range cfgs {
						if m, ok := cfg.(mockVersionedConfig); ok {
							if m.version == "test" {
								return true
							}
						}
					}
					return false
				},
			},
			cfgs:     []util.VersionedConfig{prodMock, testMock},
			expected: true,
		},
		{
			description: "does not contains a config with test version",
			s: config{
				id: "version-value-test",
				isRelevantFn: func(cfgs []util.VersionedConfig, _ sConfig.RunMode) bool {
					for _, cfg := range cfgs {
						if m, ok := cfg.(mockVersionedConfig); ok {
							if m.version == "test" {
								return true
							}
						}
					}
					return false
				},
			},
			cfgs: []util.VersionedConfig{prodMock},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.s.isRelevant(test.cfgs, "dev"), test.expected)
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		description string
		s           config
		expected    bool
	}{
		{
			description: "only hats",
			s:           hats,
			expected:    true,
		},
		{
			description: "4 weeks valid survey with start date",
			s: config{
				id:        "invalid",
				startsAt:  time.Now().AddDate(0, 1, 0),
				expiresAt: time.Now().AddDate(0, 2, 0),
			},
			expected: true,
		},
		{
			description: "4 weeks valid survey without start date",
			s: config{
				id:        "valid",
				expiresAt: time.Now().AddDate(0, 1, 0),
			},
			expected: true,
		},
		{
			description: "90 days invalid survey without start date",
			s: config{
				id:        "invalid",
				expiresAt: time.Now().AddDate(0, 0, 90),
			},
		},
		{
			description: "90 days invalid survey with start date",
			s: config{
				id:        "invalid",
				startsAt:  time.Now().AddDate(0, 1, 0),
				expiresAt: time.Now().AddDate(0, 1, 90),
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.s.isValid(), test.expected)
		})
	}
}

// mockVersionedConfig implements util.VersionedConfig.
type mockVersionedConfig struct {
	version string
}

func (m mockVersionedConfig) GetVersion() string {
	return m.version
}

func (m mockVersionedConfig) Upgrade() (util.VersionedConfig, error) {
	return m, nil
}
