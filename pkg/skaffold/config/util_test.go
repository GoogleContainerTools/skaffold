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
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestReadConfig(t *testing.T) {
	baseConfig := &GlobalConfig{
		Global: &ContextConfig{
			DefaultRepo: "test-repository",
		},
		ContextConfigs: []*ContextConfig{
			{
				Kubecontext:        "test-context",
				InsecureRegistries: []string{"bad.io", "worse.io"},
				LocalCluster:       util.BoolPtr(true),
				DefaultRepo:        "context-local-repository",
			},
		},
	}

	tests := []struct {
		description string
		filename    string
		expectedCfg *GlobalConfig
		content     *GlobalConfig
	}{
		{
			description: "first read",
			filename:    "config",
			content:     baseConfig,
			expectedCfg: baseConfig,
		},
		{
			description: "second run uses cached result",
			expectedCfg: baseConfig,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Chdir()

			if test.content != nil {
				c, _ := yaml.Marshal(*test.content)
				tmpDir.Write(test.filename, string(c))
			}

			cfg, err := ReadConfigFile(test.filename)

			t.CheckErrorAndDeepEqual(false, err, test.expectedCfg, cfg)
		})
	}
}

func TestResolveConfigFile(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		actual, err := ResolveConfigFile("")
		t.CheckNoError(err)
		suffix := filepath.FromSlash(".skaffold/config")
		if !strings.HasSuffix(actual, suffix) {
			t.Errorf("expecting %q to have suffix %q", actual, suffix)
		}
	})

	testutil.Run(t, "", func(t *testutil.T) {
		cfg := t.TempFile("givenConfigurationFile", nil)
		actual, err := ResolveConfigFile(cfg)
		t.CheckErrorAndDeepEqual(false, err, cfg, actual)
	})
}

func Test_getConfigForKubeContextWithGlobalDefaults(t *testing.T) {
	const someKubeContext = "this_is_a_context"
	sampleConfig1 := &ContextConfig{
		Kubecontext:        someKubeContext,
		InsecureRegistries: []string{"bad.io", "worse.io"},
		LocalCluster:       util.BoolPtr(true),
		DefaultRepo:        "my-private-registry",
	}
	sampleConfig2 := &ContextConfig{
		Kubecontext:  "another_context",
		LocalCluster: util.BoolPtr(false),
		DefaultRepo:  "my-public-registry",
	}

	tests := []struct {
		name           string
		kubecontext    string
		cfg            *GlobalConfig
		expectedConfig *ContextConfig
	}{
		{
			name: "global config when kubecontext is empty",
			cfg: &GlobalConfig{
				Global: &ContextConfig{
					InsecureRegistries: []string{"mediocre.io"},
					LocalCluster:       util.BoolPtr(true),
					DefaultRepo:        "my-private-registry",
				},
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: someKubeContext,
						DefaultRepo: "value",
					},
				},
			},
			expectedConfig: &ContextConfig{
				InsecureRegistries: []string{"mediocre.io"},
				LocalCluster:       util.BoolPtr(true),
				DefaultRepo:        "my-private-registry",
			},
		},
		{
			name:           "no global config and no kubecontext",
			cfg:            &GlobalConfig{},
			expectedConfig: &ContextConfig{},
		},
		{
			name:        "config for unknown kubecontext",
			kubecontext: someKubeContext,
			cfg:         &GlobalConfig{},
			expectedConfig: &ContextConfig{
				Kubecontext: someKubeContext,
			},
		},
		{
			name:        "config for kubecontext when globals are empty",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				ContextConfigs: []*ContextConfig{sampleConfig2, sampleConfig1},
			},
			expectedConfig: sampleConfig1,
		},
		{
			name:        "config for kubecontext without merged values",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				Global:         sampleConfig2,
				ContextConfigs: []*ContextConfig{sampleConfig1},
			},
			expectedConfig: sampleConfig1,
		},
		{
			name:        "config for kubecontext with merged values",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				Global: sampleConfig2,
				ContextConfigs: []*ContextConfig{
					{
						Kubecontext: someKubeContext,
					},
				},
			},
			expectedConfig: &ContextConfig{
				Kubecontext:  someKubeContext,
				LocalCluster: util.BoolPtr(false),
				DefaultRepo:  "my-public-registry",
			},
		},
		{
			name:        "config for unknown kubecontext with merged values",
			kubecontext: someKubeContext,
			cfg:         &GlobalConfig{Global: sampleConfig2},
			expectedConfig: &ContextConfig{
				Kubecontext:  someKubeContext,
				LocalCluster: util.BoolPtr(false),
				DefaultRepo:  "my-public-registry",
			},
		},
		{
			name:        "merge global and context-specific insecure-registries",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				Global: &ContextConfig{
					InsecureRegistries: []string{"good.io", "better.io"},
				},
				ContextConfigs: []*ContextConfig{{
					Kubecontext:        someKubeContext,
					InsecureRegistries: []string{"bad.io", "worse.io"},
				}},
			},
			expectedConfig: &ContextConfig{
				Kubecontext:        someKubeContext,
				InsecureRegistries: []string{"bad.io", "worse.io", "good.io", "better.io"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			actual, err := getConfigForKubeContextWithGlobalDefaults(test.cfg, test.kubecontext)
			t.CheckErrorAndDeepEqual(false, err, test.expectedConfig, actual)
		})
	}
}
