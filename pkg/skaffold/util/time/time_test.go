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

package time

import (
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLessThan(t *testing.T) {
	tests := []struct {
		description string
		date        string
		duration    time.Duration
		expected    bool
	}{
		{
			description: "date is less than 10 days from now",
			date:        time.Now().AddDate(0, 0, -5).Format(time.RFC3339),
			duration:    10 * 24 * time.Hour,
			expected:    true,
		},
		{
			description: "date is not less than 10 days from now",
			date:        time.Now().AddDate(0, 0, -11).Format(time.RFC3339),
			duration:    10 * 24 * time.Hour,
		},
		{
			description: "date is not right format",
			date:        "01-19=20129",
			expected:    false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, LessThan(test.date, test.duration))
		})
	}
}

func TestHumanize(t *testing.T) {
	duration1, err := time.ParseDuration("1h58m30.918273645s")
	if err != nil {
		t.Errorf("%s", err)
	}
	duration2, err := time.ParseDuration("5.23494327s")
	if err != nil {
		t.Errorf("%s", err)
	}
	tests := []struct {
		description string
		value       time.Duration
		expected    string
	}{
		{
			description: "Case for 1h58m30.918273645s",
			value:       duration1,
			expected:    "1 hour 58 minutes 30.918 seconds",
		},
		{
			description: "Case for 5.23494327s",
			value:       duration2,
			expected:    "5.234 seconds",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			humanizedValue := Humanize(test.value)
			t.CheckDeepEqual(test.expected, humanizedValue)
		})
	}
}
