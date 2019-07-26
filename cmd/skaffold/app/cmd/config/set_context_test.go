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
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetKubeContext(t *testing.T) {
	tests := []struct {
		name               string
		expectedCfg        *config.GlobalConfig
		skaffoldYaml       *latest.SkaffoldConfig
		configMetadataName string
		arg                string
		shouldErr          bool
	}{
		{
			name:        "insufficient arguments",
			arg:         "this_is_a_context",
			expectedCfg: &config.GlobalConfig{},
			shouldErr:   true,
		},
		{
			name:               "explicit config-name",
			arg:                "this_is_a_context",
			configMetadataName: "my-skaffold-config",
			expectedCfg: &config.GlobalConfig{
				SkaffoldConfigs: map[string]string{"my-skaffold-config": "this_is_a_context"},
				ContextConfigs:  []*config.ContextConfig{},
			},
		},
		{
			name:               "cannot specify --filename and --skaffold-config",
			arg:                "this_is_a_context",
			configMetadataName: "name-on-cli",
			skaffoldYaml: &latest.SkaffoldConfig{
				Metadata: latest.Metadata{
					Name: "name-in-skaffold.yaml",
				},
			},
			expectedCfg: &config.GlobalConfig{},
			shouldErr:   true,
		},
		{
			name: "take pipeline name from skaffold.yaml",
			arg:  "this_is_a_context",
			skaffoldYaml: &latest.SkaffoldConfig{
				APIVersion: latest.Version,
				Kind:       "Config",
				Metadata:   latest.Metadata{Name: "name-in-skaffold.yaml"},
			},
			expectedCfg: &config.GlobalConfig{
				SkaffoldConfigs: map[string]string{"name-in-skaffold.yaml": "this_is_a_context"},
				ContextConfigs:  []*config.ContextConfig{},
			},
		},
		{
			name: "skaffold.yaml without a name",
			arg:  "this_is_a_context",
			skaffoldYaml: &latest.SkaffoldConfig{
				APIVersion: latest.Version,
				Kind:       "Config",
				Metadata:   latest.Metadata{Name: ""},
			},
			expectedCfg: &config.GlobalConfig{},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			// create new config file
			cfg := t.TempFile("config", nil)

			t.Override(&config.ReadConfigFile, config.ReadConfigFileNoCache)
			t.Override(&configFile, cfg)
			t.Override(&configMetadataName, test.configMetadataName)

			if test.skaffoldYaml != nil {
				c, _ := yaml.Marshal(*test.skaffoldYaml)
				skaffoldYaml := t.TempFile("skaffold-yaml", c)
				t.Override(&skaffoldYamlFile, skaffoldYaml)
			}

			// set specified value
			err := SetKubeContext(ioutil.Discard, []string{test.arg})
			actualConfig, cfgErr := config.ReadConfigFile(cfg)
			t.CheckNoError(cfgErr)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedCfg, actualConfig)
		})
	}
}
