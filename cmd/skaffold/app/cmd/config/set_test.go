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

package config

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetAndUnsetConfig(t *testing.T) {
	dummyContext := "dummy_context"

	tests := []struct {
		expectedSetCfg   *config.GlobalConfig
		expectedUnsetCfg *config.GlobalConfig
		description      string
		key              string
		value            string
		kubecontext      string
		global           bool
		survey           bool
		shouldErr        bool
	}{
		{
			description: "set default repo",
			key:         "default-repo",
			value:       "value",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext: "this_is_a_context",
						DefaultRepo: "value",
					},
				},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
		{
			description: "set local cluster",
			key:         "local-cluster",
			value:       "false",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext:  "this_is_a_context",
						LocalCluster: util.BoolPtr(false),
					},
				},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
		{
			description:    "set invalid local cluster",
			key:            "local-cluster",
			shouldErr:      true,
			value:          "not-a-bool",
			expectedSetCfg: &config.GlobalConfig{},
		},
		{
			description:    "set fake value",
			key:            "not_a_real_value",
			shouldErr:      true,
			expectedSetCfg: &config.GlobalConfig{},
		},
		{
			description: "set global default repo",
			key:         "default-repo",
			value:       "global",
			global:      true,
			expectedSetCfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					DefaultRepo: "global",
				},
				ContextConfigs: []*config.ContextConfig{},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				Global:         &config.ContextConfig{},
				ContextConfigs: []*config.ContextConfig{},
			},
		},
		{
			description: "set global local cluster",
			key:         "local-cluster",
			value:       "true",
			global:      true,
			expectedSetCfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					LocalCluster: util.BoolPtr(true),
				},
				ContextConfigs: []*config.ContextConfig{},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				Global:         &config.ContextConfig{},
				ContextConfigs: []*config.ContextConfig{},
			},
		},
		{
			description: "set insecure registries",
			key:         "insecure-registries",
			value:       "my.insecure.registry",
			kubecontext: "this_is_a_context",
			expectedSetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext:        "this_is_a_context",
						InsecureRegistries: []string{"my.insecure.registry"},
					},
				},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext: "this_is_a_context",
					},
				},
			},
		},
		{
			description: "set global update-check",
			key:         "update-check",
			value:       "true",
			global:      true,
			expectedSetCfg: &config.GlobalConfig{
				Global:         &config.ContextConfig{UpdateCheck: util.BoolPtr(true)},
				ContextConfigs: []*config.ContextConfig{},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				Global:         &config.ContextConfig{},
				ContextConfigs: []*config.ContextConfig{},
			},
		},
		{
			description: "set global survey disable prompt",
			key:         "disable-prompt",
			value:       "true",
			global:      true,
			survey:      true,
			expectedSetCfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					Survey: &config.SurveyConfig{
						DisablePrompt: util.BoolPtr(true),
					},
				},
				ContextConfigs: []*config.ContextConfig{},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				Global:         &config.ContextConfig{Survey: &config.SurveyConfig{}},
				ContextConfigs: []*config.ContextConfig{},
			},
		},
		{
			description: "set global survey disable prompt false",
			key:         "disable-prompt",
			value:       "false",
			global:      true,
			survey:      true,
			expectedSetCfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					Survey: &config.SurveyConfig{
						DisablePrompt: util.BoolPtr(false),
					},
				},
				ContextConfigs: []*config.ContextConfig{},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				Global: &config.ContextConfig{
					Survey: &config.SurveyConfig{},
				},
				ContextConfigs: []*config.ContextConfig{},
			},
		},
		{
			description: "set survey disable prompt",
			key:         "disable-prompt",
			value:       "false",
			kubecontext: "this_is_a_context",
			survey:      true,
			expectedSetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext: "this_is_a_context",
						Survey: &config.SurveyConfig{
							DisablePrompt: util.BoolPtr(false),
						},
					},
				},
			},
			expectedUnsetCfg: &config.GlobalConfig{
				ContextConfigs: []*config.ContextConfig{
					{
						Kubecontext: "this_is_a_context",
						Survey:      &config.SurveyConfig{},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// create new config file
			cfg := t.TempFile("config", nil)

			t.Override(&config.ReadConfigFile, config.ReadConfigFileNoCache)
			t.Override(&configFile, cfg)
			t.Override(&global, test.global)
			t.Override(&survey, test.survey)
			if test.kubecontext != "" {
				t.Override(&kubecontext, test.kubecontext)
			} else {
				t.Override(&kubecontext, dummyContext)
			}

			// set specified value
			err := Set(context.Background(), ioutil.Discard, []string{test.key, test.value})
			actualConfig, cfgErr := config.ReadConfigFile(cfg)
			t.CheckNoError(cfgErr)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedSetCfg, actualConfig)

			if test.shouldErr {
				// if we expect an error when setting, don't try to unset
				return
			}

			// unset the value
			err = Unset(context.Background(), ioutil.Discard, []string{test.key})
			newConfig, cfgErr := config.ReadConfigFile(cfg)
			t.CheckNoError(cfgErr)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedUnsetCfg, newConfig)
		})
	}
}

func TestGetConfigStructWithIndex(t *testing.T) {
	tests := []struct {
		description string
		cfg         *config.ContextConfig
		expectedIdx []int
		survey      bool
		shouldErr   bool
	}{
		{
			description: "survey flag set",
			cfg:         &config.ContextConfig{},
			survey:      true,
			expectedIdx: []int{6},
		},
		{
			description: "no survey flag set",
			cfg:         &config.ContextConfig{},
			expectedIdx: []int{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&survey, test.survey)
			actual, err := getConfigStructWithIndex(test.cfg)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedIdx, actual.idx)
		})
	}
}
