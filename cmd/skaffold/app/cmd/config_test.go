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

package cmd

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var baseConfig = &config.Config{
	Global: &config.ContextConfig{
		DefaultRepo: "test-repository",
	},
	ContextConfigs: []*config.ContextConfig{
		{
			Kubecontext: "test-context",
			DefaultRepo: "context-local-repository",
		},
	},
}

func TestReadConfig(t *testing.T) {
	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	var tests = []struct {
		filename    string
		expectedCfg *config.Config
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
		cfg, err := config.ReadConfigForFile(test.filename)

		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedCfg, cfg)
	}
}
