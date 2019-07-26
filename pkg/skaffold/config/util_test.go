package config

import (
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
		const suffix = ".skaffold/config"
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
		Kubecontext:        "another_context",
		InsecureRegistries: []string{"good.io", "better.io"},
		LocalCluster:       util.BoolPtr(false),
		DefaultRepo:        "my-public-registry",
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
				Kubecontext:        someKubeContext,
				InsecureRegistries: []string{"good.io", "better.io"},
				LocalCluster:       util.BoolPtr(false),
				DefaultRepo:        "my-public-registry",
			},
		},
		{
			name:        "config for unknown kubecontext with merged values",
			kubecontext: someKubeContext,
			cfg:         &GlobalConfig{Global: sampleConfig2},
			expectedConfig: &ContextConfig{
				Kubecontext:        someKubeContext,
				InsecureRegistries: []string{"good.io", "better.io"},
				LocalCluster:       util.BoolPtr(false),
				DefaultRepo:        "my-public-registry",
			},
		},
		/* todo(corneliusweig): this behavior can be enabled with `mergo.WithAppendSlice` -> clarify requirements
		{
			name:        "merge global and context-specific insecure-registries",
			kubecontext: someKubeContext,
			cfg: &GlobalConfig{
				Global:         &ContextConfig{
					InsecureRegistries: []string{"good.io", "better.io"},
				},
				ContextConfigs: []*ContextConfig{{
					Kubecontext: someKubeContext,
					InsecureRegistries: []string{"bad.io", "worse.io"},
				}},
			},
			expectedConfig: &ContextConfig{
				Kubecontext:        someKubeContext,
				InsecureRegistries: []string{"bad.io", "worse.io", "good.io", "better.io"},
			},
		},*/
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			actual, err := getConfigForKubeContextWithGlobalDefaults(test.cfg, test.kubecontext)
			t.CheckErrorAndDeepEqual(false, err, test.expectedConfig, actual)
		})
	}
}
