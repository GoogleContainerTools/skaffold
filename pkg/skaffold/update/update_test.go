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

package update

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsUpdateCheckEnabledByEnvOrConfig(t *testing.T) {
	tests := []struct {
		description string
		envVariable string
		cfg         *config.GlobalConfig
		readErr     error
		expected    bool
	}{
		{
			description: "nothing is set opt-in behavior",
			expected:    true,
		}, {
			description: "env variable is set to true",
			envVariable: "true",
			expected:    true,
		}, {
			description: "env variable is set to false",
			envVariable: "false",
		}, {
			description: "env variable is set to random string",
			envVariable: "foo",
		}, {
			description: "env variable is empty Global config true",
			cfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					UpdateCheck: util.BoolPtr(true),
				},
			},
			expected: true,
		}, {
			description: "env variable is empty Global config false",
			cfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					UpdateCheck: util.BoolPtr(false),
				},
			},
			expected: false,
		}, {
			description: "env variable and Global config update-check is empty - opt-in ",
			cfg: &config.GlobalConfig{
				Global: &config.ContextConfig{},
			},
			expected: true,
		}, {
			description: "error when reading config",
			readErr:     fmt.Errorf("reading error"),
			expected:    true,
		}, {
			description: "env variable is false but Global config update-check is true",
			envVariable: "false",
			cfg: &config.GlobalConfig{
				Global: &config.ContextConfig{UpdateCheck: util.BoolPtr(true)},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&readConfig, func(string) (*config.GlobalConfig, error) { return test.cfg, test.readErr })
			t.Override(&getEnv, func(string) string { return test.envVariable })
			t.CheckDeepEqual(test.expected, isUpdateCheckEnabledByEnvOrConfig("dummyconfig"))
		})
	}
}
