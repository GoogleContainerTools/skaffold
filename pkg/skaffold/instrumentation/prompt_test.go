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

package instrumentation

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestShouldDisplayMetricsPrompt(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.ContextConfig
		expected bool
		err      error
	}{
		{
			name:     "empty config",
			config:   &config.ContextConfig{},
			expected: true,
		},
		{
			name:     "nil config",
			expected: true,
		},
		{
			name: "not nil error",
			err:  fmt.Errorf("test error getting config"),
		},
		{
			name:     "config without collect-metrics",
			config:   &config.ContextConfig{DefaultRepo: "test-repo"},
			expected: true,
		},
		{
			name:   "collect-metrics false",
			config: &config.ContextConfig{CollectMetrics: boolP(false)},
		},
		{
			name:   "collect-metrics true",
			config: &config.ContextConfig{CollectMetrics: boolP(true)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			mock := func(string) (*config.ContextConfig, error) { return test.config, test.err }
			t.Override(&getConfig, mock)
			t.Override(&setStatus, func() {})
			actual := ShouldDisplayMetricsPrompt(test.name)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestDisplayMetricsPrompt(t *testing.T) {
	tests := []struct {
		name       string
		mockStdOut bool
		expected   string
	}{
		{
			name:       "std out",
			mockStdOut: true,
			expected:   Prompt,
		},
		{
			name: "not std out",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			mock := func(io.Writer) bool { return test.mockStdOut }
			t.Override(&isStdOut, mock)
			t.Override(&updateConfig, func(_ string, _ bool) error { return nil })
			var buf bytes.Buffer
			err := DisplayMetricsPrompt("", &buf)
			t.CheckErrorAndDeepEqual(false, err, test.expected, buf.String())
		})
	}
}

func boolP(b bool) *bool { return &b }
