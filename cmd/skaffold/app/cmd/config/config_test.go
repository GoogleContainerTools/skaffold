/*
Copyright 2018 The Skaffold Authors

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

package config

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

var baseConfig = &Config{
	Global: &ContextConfig{
		DefaultRepo: "test-repository",
	},
	ContextConfigs: []*ContextConfig{
		{
			Kubecontext: "test-context",
			DefaultRepo: "context-local-repository",
		},
	},
}

var emptyConfig = &Config{}

func TestReadConfig(t *testing.T) {
	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	var tests = []struct {
		filename    string
		expectedCfg *Config
		shouldErr   bool
	}{
		{
			filename:  "",
			shouldErr: true,
		},
		{
			filename:    cfg,
			expectedCfg: baseConfig,
			shouldErr:   false,
		},
	}

	for _, test := range tests {
		cfg, err := ReadConfigForFile(test.filename)

		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedCfg, cfg)
	}
}

func TestSetAndUnsetConfig(t *testing.T) {
	var tests = []struct {
		expectedSetCfg   *Config
		expectedUnsetCfg *Config
		name             string
		key              string
		value            string
		kubecontext      string
		global           bool
		shouldErrSet     bool
	}{
		{
			name:        "set default repo",
			key:         "default-repo",
			value:       "value",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: "this_is_a_context",
						DefaultRepo: "value",
					},
				},
			},
			expectedUnsetCfg: &Config{
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
		{
			name:         "set fake value",
			key:          "not_a_real_value",
			shouldErrSet: true,
			expectedSetCfg: &Config{
				ContextConfigs: []*ContextConfig{{}},
			},
		},
		{
			name:   "set global default repo",
			key:    "default-repo",
			value:  "global",
			global: true,
			expectedSetCfg: &Config{
				Global: &ContextConfig{
					DefaultRepo: "global",
				},
				ContextConfigs: []*ContextConfig{},
			},
			expectedUnsetCfg: &Config{
				Global:         &ContextConfig{},
				ContextConfigs: []*ContextConfig{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create new config file
			c, _ := yaml.Marshal(*emptyConfig)
			cfg, teardown := testutil.TempFile(t, "config", c)

			// setup config context
			kubecontext = test.kubecontext
			configFile = cfg
			global = test.global

			// set specified value
			err := setConfigValue(test.key, test.value)
			actualConfig, cfgErr := readConfig()
			if cfgErr != nil {
				t.Error(cfgErr)
			}
			testutil.CheckErrorAndDeepEqual(t, test.shouldErrSet, err, test.expectedSetCfg, actualConfig)

			if test.shouldErrSet {
				// if we expect an error when setting, don't try and unset
				return
			}

			// unset the value
			err = unsetConfigValue(test.key)
			newConfig, cfgErr := readConfig()
			if cfgErr != nil {
				t.Error(cfgErr)
			}
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedUnsetCfg, newConfig)

			// reset all config context for next test
			teardown()
			kubecontext = ""
			configFile = ""
			global = false
		})
	}
}
