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

package prompt

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/AlecAivazis/survey/v2"

	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestWriteSkaffoldConfig(t *testing.T) {
	tests := []struct {
		description    string
		config         *latest_v1.SkaffoldConfig
		promptResponse bool
		expectedDone   bool
		shouldErr      bool
	}{
		{
			description:    "yes response",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: true,
			expectedDone:   false,
			shouldErr:      false,
		},
		{
			description:    "no response",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: false,
			expectedDone:   true,
			shouldErr:      false,
		},
		{
			description:    "error",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: false,
			expectedDone:   true,
			shouldErr:      true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&askOne, func(_ survey.Prompt, response interface{}, _ ...survey.AskOpt) error {
				r := response.(*bool)
				*r = test.promptResponse

				if test.shouldErr {
					return errors.New("error")
				}
				return nil
			})

			done, err := WriteSkaffoldConfig(ioutil.Discard, []byte{}, nil, "")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDone, done)
		})
	}
}

func TestChooseBuilders(t *testing.T) {
	tests := []struct {
		description    string
		choices        []string
		promptResponse []string
		expected       []string
		shouldErr      bool
	}{
		{
			description:    "couple chosen",
			choices:        []string{"a", "b", "c"},
			promptResponse: []string{"a", "c"},
			expected:       []string{"a", "c"},
			shouldErr:      false,
		},
		{
			description:    "none chosen",
			choices:        []string{"a", "b", "c"},
			promptResponse: []string{},
			expected:       []string{},
			shouldErr:      false,
		},
		{
			description:    "error",
			choices:        []string{"a", "b", "c"},
			promptResponse: []string{"a", "b"},
			expected:       []string{},
			shouldErr:      true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&askOne, func(_ survey.Prompt, response interface{}, _ ...survey.AskOpt) error {
				r := response.(*[]string)
				*r = test.promptResponse

				if test.shouldErr {
					return errors.New("error")
				}
				return nil
			})

			chosen, err := ChooseBuildersFunc(test.choices)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, chosen)
		})
	}
}

func TestPortForwardResource(t *testing.T) {
	tests := []struct {
		description    string
		config         *latest_v1.SkaffoldConfig
		promptResponse string
		expected       int
		shouldErr      bool
	}{
		{
			description:    "valid response",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: "8080",
			expected:       8080,
			shouldErr:      false,
		},
		{
			description:    "empty response",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: "",
			expected:       0,
			shouldErr:      false,
		},
		{
			description:    "error",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: "",
			expected:       0,
			shouldErr:      true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&ask, func(_ []*survey.Question, response interface{}, _ ...survey.AskOpt) error {
				r := response.(*string)
				*r = test.promptResponse

				if test.shouldErr {
					return errors.New("error")
				}
				return nil
			})

			port, err := portForwardResource(ioutil.Discard, "image-name")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, port)
		})
	}
}

func TestConfirmInitOptions(t *testing.T) {
	tests := []struct {
		description    string
		config         *latest_v1.SkaffoldConfig
		promptResponse bool
		expectedDone   bool
		shouldErr      bool
	}{
		{
			description:    "yes response",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: true,
			expectedDone:   false,
			shouldErr:      false,
		},
		{
			description:    "no response",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: false,
			expectedDone:   true,
			shouldErr:      false,
		},
		{
			description:    "error",
			config:         &latest_v1.SkaffoldConfig{},
			promptResponse: false,
			expectedDone:   true,
			shouldErr:      true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&askOne, func(_ survey.Prompt, response interface{}, _ ...survey.AskOpt) error {
				r := response.(*bool)
				*r = test.promptResponse

				if test.shouldErr {
					return errors.New("error")
				}
				return nil
			})

			done, err := ConfirmInitOptions(ioutil.Discard, test.config)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDone, done)
		})
	}
}
