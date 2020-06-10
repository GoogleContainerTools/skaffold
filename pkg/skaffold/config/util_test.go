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
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
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

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedCfg, cfg)
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
		t.CheckNoError(err)
		t.CheckDeepEqual(cfg, actual)
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
		description    string
		kubecontext    string
		cfg            *GlobalConfig
		expectedConfig *ContextConfig
	}{
		{
			description: "global config when kubecontext is empty",
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
			description:    "no global config and no kubecontext",
			cfg:            &GlobalConfig{},
			expectedConfig: &ContextConfig{},
		},
		{
			description: "config for unknown kubecontext",
			kubecontext: someKubeContext,
			cfg:         &GlobalConfig{},
			expectedConfig: &ContextConfig{
				Kubecontext: someKubeContext,
			},
		},
		{
			description: "config for kubecontext when globals are empty",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				ContextConfigs: []*ContextConfig{sampleConfig2, sampleConfig1},
			},
			expectedConfig: sampleConfig1,
		},
		{
			description: "config for kubecontext without merged values",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				Global:         sampleConfig2,
				ContextConfigs: []*ContextConfig{sampleConfig1},
			},
			expectedConfig: sampleConfig1,
		},
		{
			description: "config for kubecontext with merged values",
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
			description: "config for unknown kubecontext with merged values",
			kubecontext: someKubeContext,
			cfg:         &GlobalConfig{Global: sampleConfig2},
			expectedConfig: &ContextConfig{
				Kubecontext:  someKubeContext,
				LocalCluster: util.BoolPtr(false),
				DefaultRepo:  "my-public-registry",
			},
		},
		{
			description: "merge global and context-specific insecure-registries",
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
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := getConfigForKubeContextWithGlobalDefaults(test.cfg, test.kubecontext)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedConfig, actual)
		})
	}
}

func TestIsUpdateCheckEnabled(t *testing.T) {
	tests := []struct {
		description string
		cfg         *ContextConfig
		readErr     error
		expected    bool
	}{
		{
			description: "config update-check is nil returns true",
			cfg:         &ContextConfig{},
			expected:    true,
		},
		{
			description: "config update-check is true",
			cfg:         &ContextConfig{UpdateCheck: util.BoolPtr(true)},
			expected:    true,
		},
		{
			description: "config update-check is false",
			cfg:         &ContextConfig{UpdateCheck: util.BoolPtr(false)},
		},
		{
			description: "config is nil",
			cfg:         nil,
			expected:    true,
		},
		{
			description: "config has err",
			cfg:         nil,
			readErr:     fmt.Errorf("error while reading"),
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, test.readErr })
			actual := IsUpdateCheckEnabled("dummyconfig")
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestIsDefaultLocal(t *testing.T) {
	tests := []struct {
		context       string
		expectedLocal bool
	}{
		{context: "kind-other", expectedLocal: true},
		{context: "kind@kind", expectedLocal: true},
		{context: "k3d-k3s-default", expectedLocal: true},
		{context: "docker-for-desktop", expectedLocal: true},
		{context: "minikube", expectedLocal: true},
		{context: "docker-desktop", expectedLocal: true},
		{context: "anything-else", expectedLocal: false},
		{context: "kind@blah", expectedLocal: false},
		{context: "other-kind", expectedLocal: false},
		{context: "not-k3d", expectedLocal: false},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			local := isDefaultLocal(test.context)

			t.CheckDeepEqual(test.expectedLocal, local)
		})
	}
}

func TestIsImageLoadingRequired(t *testing.T) {
	tests := []struct {
		context                      string
		expectedImageLoadingRequired bool
	}{
		{context: "kind-other", expectedImageLoadingRequired: true},
		{context: "kind@kind", expectedImageLoadingRequired: true},
		{context: "k3d-k3s-default", expectedImageLoadingRequired: true},
		{context: "docker-for-desktop", expectedImageLoadingRequired: false},
		{context: "minikube", expectedImageLoadingRequired: false},
		{context: "docker-desktop", expectedImageLoadingRequired: false},
		{context: "anything-else", expectedImageLoadingRequired: false},
		{context: "kind@blah", expectedImageLoadingRequired: false},
		{context: "other-kind", expectedImageLoadingRequired: false},
		{context: "not-k3d", expectedImageLoadingRequired: false},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			imageLoadingRequired := IsImageLoadingRequired(test.context)

			t.CheckDeepEqual(test.expectedImageLoadingRequired, imageLoadingRequired)
		})
	}
}

func TestIsKindCluster(t *testing.T) {
	tests := []struct {
		context        string
		expectedIsKind bool
	}{
		{context: "kind-kind", expectedIsKind: true},
		{context: "kind-other", expectedIsKind: true},
		{context: "kind@kind", expectedIsKind: true},
		{context: "other@kind", expectedIsKind: true},
		{context: "docker-for-desktop", expectedIsKind: false},
		{context: "not-kind", expectedIsKind: false},
	}
	for _, test := range tests {
		testutil.Run(t, test.context, func(t *testutil.T) {
			isKind := IsKindCluster(test.context)

			t.CheckDeepEqual(test.expectedIsKind, isKind)
		})
	}
}

func TestKindClusterName(t *testing.T) {
	tests := []struct {
		kubeCluster  string
		expectedName string
	}{
		{kubeCluster: "kind", expectedName: "kind"},
		{kubeCluster: "kind-kind", expectedName: "kind"},
		{kubeCluster: "kind-other", expectedName: "other"},
		{kubeCluster: "kind@kind", expectedName: "kind"},
		{kubeCluster: "other@kind", expectedName: "other"},
	}
	for _, test := range tests {
		testutil.Run(t, test.kubeCluster, func(t *testutil.T) {
			kindCluster := KindClusterName(test.kubeCluster)

			t.CheckDeepEqual(test.expectedName, kindCluster)
		})
	}
}

func TestIsK3dCluster(t *testing.T) {
	tests := []struct {
		context       string
		expectedIsK3d bool
	}{
		{context: "k3d-k3s-default", expectedIsK3d: true},
		{context: "k3d-other", expectedIsK3d: true},
		{context: "kind-kind", expectedIsK3d: false},
		{context: "docker-for-desktop", expectedIsK3d: false},
		{context: "not-k3d", expectedIsK3d: false},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			isK3d := IsK3dCluster(test.context)

			t.CheckDeepEqual(test.expectedIsK3d, isK3d)
		})
	}
}

func TestK3dClusterName(t *testing.T) {
	tests := []struct {
		kubeCluster  string
		expectedName string
	}{
		{kubeCluster: "k3d-k3s-default", expectedName: "k3s-default"},
		{kubeCluster: "k3d-other", expectedName: "other"},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			k3dCluster := K3dClusterName(test.kubeCluster)

			t.CheckDeepEqual(test.expectedName, k3dCluster)
		})
	}
}

func TestIsSurveyPromptDisabled(t *testing.T) {
	tests := []struct {
		description string
		cfg         *ContextConfig
		readErr     error
		expected    bool
	}{
		{
			description: "config disable-prompt is nil returns false",
			cfg:         &ContextConfig{},
		},
		{
			description: "config disable-prompt is true",
			cfg:         &ContextConfig{Survey: &SurveyConfig{DisablePrompt: util.BoolPtr(true)}},
			expected:    true,
		},
		{
			description: "config disable-prompt is false",
			cfg:         &ContextConfig{Survey: &SurveyConfig{DisablePrompt: util.BoolPtr(false)}},
		},
		{
			description: "disable prompt is nil",
			cfg:         &ContextConfig{Survey: &SurveyConfig{}},
		},
		{
			description: "config is nil",
			cfg:         nil,
		},
		{
			description: "config has err",
			cfg:         nil,
			readErr:     fmt.Errorf("error while reading"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, test.readErr })
			_, actual := isSurveyPromptDisabled("dummyconfig")
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		description string
		date        string
		duration    time.Duration
		expected    bool
	}{
		{
			description: "date is less than 10 days from 01/30/2019",
			date:        "2019-01-22T13:04:05Z",
			duration:    10 * 24 * time.Hour,
			expected:    true,
		},
		{
			description: "date is not less than 10 days from 01/30/2019",
			date:        "2019-01-19T13:04:05Z",
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
			t.Override(&current, func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2019-01-30T12:04:05Z")
				return t
			})
			t.CheckDeepEqual(test.expected, lessThan(test.date, test.duration))
		})
	}
}

func TestShouldDisplayPrompt(t *testing.T) {
	tests := []struct {
		description string
		cfg         *ContextConfig
		expected    bool
	}{
		{
			description: "should not display prompt when prompt is disabled",
			cfg:         &ContextConfig{Survey: &SurveyConfig{DisablePrompt: util.BoolPtr(true)}},
		},
		{
			description: "should not display prompt when last prompted is less than 2 weeks",
			cfg: &ContextConfig{
				Survey: &SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastPrompted:  "2019-01-22T00:00:00Z",
				},
			},
		},
		{
			description: "should not display prompt when last taken in less than 3 months",
			cfg: &ContextConfig{
				Survey: &SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     "2018-11-22T00:00:00Z",
				},
			},
		},
		{
			description: "should display prompt when last prompted is before 2 weeks",
			cfg: &ContextConfig{
				Survey: &SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastPrompted:  "2019-01-10T00:00:00Z",
				},
			},
			expected: true,
		},
		{
			description: "should display prompt when last taken is before than 3 months ago",
			cfg: &ContextConfig{
				Survey: &SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     "2017-11-10T00:00:00Z",
				},
			},
			expected: true,
		},
		{
			description: "should not display prompt when last taken is recent than 3 months ago",
			cfg: &ContextConfig{
				Survey: &SurveyConfig{
					DisablePrompt: util.BoolPtr(false),
					LastTaken:     "2019-01-10T00:00:00Z",
					LastPrompted:  "2019-01-10T00:00:00Z",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, nil })
			t.Override(&current, func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2019-01-30T12:04:05Z")
				return t
			})
			t.CheckDeepEqual(test.expected, ShouldDisplayPrompt("dummyconfig"))
		})
	}
}

func TestGetDefaultRepo(t *testing.T) {
	tests := []struct {
		description  string
		cfg          *ContextConfig
		cliValue     *string
		expectedRepo string
		shouldErr    bool
	}{
		{
			description:  "empty",
			cfg:          &ContextConfig{},
			cliValue:     nil,
			expectedRepo: "",
		},
		{
			description:  "from cli",
			cfg:          &ContextConfig{},
			cliValue:     util.StringPtr("default/repo"),
			expectedRepo: "default/repo",
		},
		{
			description:  "from global config",
			cfg:          &ContextConfig{DefaultRepo: "global/repo"},
			cliValue:     nil,
			expectedRepo: "global/repo",
		},
		{
			description:  "cancel global config with cli",
			cfg:          &ContextConfig{DefaultRepo: "global/repo"},
			cliValue:     util.StringPtr(""),
			expectedRepo: "",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, nil })

			defaultRepo, err := GetDefaultRepo("config", test.cliValue)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedRepo, defaultRepo)
		})
	}
}

func TestUpdateGlobalSurveyTaken(t *testing.T) {
	tests := []struct {
		description string
		cfg         string
		expectedCfg *GlobalConfig
	}{
		{
			description: "update global context when context is empty",
			expectedCfg: &GlobalConfig{
				Global:         &ContextConfig{Survey: &SurveyConfig{}},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "update global context when survey config is not nil",
			cfg: `
global:
  survey:
    last-prompted: "some date"
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global:         &ContextConfig{Survey: &SurveyConfig{LastPrompted: "some date"}},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "update global context when survey config last taken is in past",
			cfg: `
global:
  survey:
    last-taken: "some date in past"
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global:         &ContextConfig{Survey: &SurveyConfig{}},
				ContextConfigs: []*ContextConfig{},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := t.TempFile("config", []byte(test.cfg))
			testTime := time.Now()
			t.Override(&ReadConfigFile, ReadConfigFileNoCache)
			t.Override(&current, func() time.Time {
				return testTime
			})

			// update the time
			err := UpdateGlobalSurveyTaken(cfg)
			t.CheckNoError(err)

			actualConfig, cfgErr := ReadConfigFile(cfg)
			t.CheckNoError(cfgErr)
			// update time in expected cfg.
			test.expectedCfg.Global.Survey.LastTaken = testTime.Format(time.RFC3339)
			t.CheckDeepEqual(test.expectedCfg, actualConfig)
		})
	}
}

func TestUpdateGlobalSurveyPrompted(t *testing.T) {
	tests := []struct {
		description string
		cfg         string
		expectedCfg *GlobalConfig
	}{
		{
			description: "update global context when context is empty",
			expectedCfg: &GlobalConfig{
				Global:         &ContextConfig{Survey: &SurveyConfig{}},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "update global context when survey config is not nil",
			cfg: `
global:
  survey:
    last-taken: "some date"
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global:         &ContextConfig{Survey: &SurveyConfig{LastTaken: "some date"}},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "update global context when survey config last prompted is in past",
			cfg: `
global:
  survey:
    last-prompted: "some date in past"
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global:         &ContextConfig{Survey: &SurveyConfig{}},
				ContextConfigs: []*ContextConfig{},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := t.TempFile("config", []byte(test.cfg))
			testTime := time.Now()
			t.Override(&ReadConfigFile, ReadConfigFileNoCache)
			t.Override(&current, func() time.Time {
				return testTime
			})

			// update the time
			err := UpdateGlobalSurveyPrompted(cfg)
			t.CheckNoError(err)

			actualConfig, cfgErr := ReadConfigFile(cfg)
			t.CheckNoError(cfgErr)
			// update time in expected cfg.
			test.expectedCfg.Global.Survey.LastPrompted = testTime.Format(time.RFC3339)
			t.CheckDeepEqual(test.expectedCfg, actualConfig)
		})
	}
}
