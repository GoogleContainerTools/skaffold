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
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/cluster"
	kubeclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestReadConfig(t *testing.T) {
	baseConfig := &GlobalConfig{
		Global: &ContextConfig{
			DefaultRepo:    "test-repository",
			MultiLevelRepo: util.Ptr(false),
		},
		ContextConfigs: []*ContextConfig{
			{
				Kubecontext:        "test-context",
				InsecureRegistries: []string{"bad.io", "worse.io"},
				LocalCluster:       util.Ptr(true),
				DefaultRepo:        "context-local-repository",
				MultiLevelRepo:     util.Ptr(true),
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
		LocalCluster:       util.Ptr(true),
		DefaultRepo:        "my-private-registry",
		MultiLevelRepo:     util.Ptr(true),
	}
	sampleConfig2 := &ContextConfig{
		Kubecontext:    "another_context",
		LocalCluster:   util.Ptr(false),
		DefaultRepo:    "my-public-registry",
		MultiLevelRepo: util.Ptr(false),
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
					LocalCluster:       util.Ptr(true),
					DefaultRepo:        "my-private-registry",
					MultiLevelRepo:     util.Ptr(true),
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
				LocalCluster:       util.Ptr(true),
				DefaultRepo:        "my-private-registry",
				MultiLevelRepo:     util.Ptr(true),
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
				Kubecontext:    someKubeContext,
				LocalCluster:   util.Ptr(false),
				DefaultRepo:    "my-public-registry",
				MultiLevelRepo: util.Ptr(false),
			},
		},
		{
			description: "config for unknown kubecontext with merged values",
			kubecontext: someKubeContext,
			cfg:         &GlobalConfig{Global: sampleConfig2},
			expectedConfig: &ContextConfig{
				Kubecontext:    someKubeContext,
				LocalCluster:   util.Ptr(false),
				DefaultRepo:    "my-public-registry",
				MultiLevelRepo: util.Ptr(false),
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
			cfg:         &ContextConfig{UpdateCheck: util.Ptr(true)},
			expected:    true,
		},
		{
			description: "config update-check is false",
			cfg:         &ContextConfig{UpdateCheck: util.Ptr(false)},
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

type fakeClient struct{}

func (fakeClient) IsMinikube(ctx context.Context, kubeContext string) bool {
	return kubeContext == "minikube"
}
func (fakeClient) MinikubeExec(context.Context, ...string) (*exec.Cmd, error) { return nil, nil }

func TestGetCluster(t *testing.T) {
	var defaultRepo = "localhost:4000"

	tests := []struct {
		description string
		cfg         *ContextConfig
		defaultRepo StringOrUndefined
		registry    *string
		profile     string
		expected    Cluster
	}{
		{
			description: "kind",
			cfg:         &ContextConfig{Kubecontext: "kind-other"},
			expected:    Cluster{Local: true, LoadImages: true, PushImages: false},
		},
		{
			description: "kind with registry config",
			cfg:         &ContextConfig{Kubecontext: "kind-other"},
			registry:    &defaultRepo,
			expected:    Cluster{Local: true, LoadImages: false, PushImages: true, DefaultRepo: NewStringOrUndefined(&defaultRepo)},
		},
		{
			description: "kind with default-repo=localhost:4000",
			cfg:         &ContextConfig{Kubecontext: "kind-other"},
			defaultRepo: NewStringOrUndefined(&defaultRepo),
			expected:    Cluster{Local: true, LoadImages: true, PushImages: false, DefaultRepo: NewStringOrUndefined(&defaultRepo)},
		},
		{
			description: "kind with local-cluster=false",
			cfg:         &ContextConfig{Kubecontext: "kind-other", LocalCluster: util.Ptr(false)},
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true},
		},
		{
			description: "kind with kind-disable-load=true",
			cfg:         &ContextConfig{Kubecontext: "kind-other", KindDisableLoad: util.Ptr(true)},
			expected:    Cluster{Local: true, LoadImages: false, PushImages: true},
		},
		{
			description: "kind with legacy name",
			cfg:         &ContextConfig{Kubecontext: "kind@kind"},
			expected:    Cluster{Local: true, LoadImages: true, PushImages: false},
		},
		{
			description: "k3d",
			cfg:         &ContextConfig{Kubecontext: "k3d-k3s-default"},
			expected:    Cluster{Local: true, LoadImages: true, PushImages: false},
		},
		{
			description: "k3d with local-cluster=false",
			cfg:         &ContextConfig{Kubecontext: "k3d-k3s-default", LocalCluster: util.Ptr(false)},
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true},
		},
		{
			description: "k3d with disable-load=true",
			cfg:         &ContextConfig{Kubecontext: "k3d-k3s-default", K3dDisableLoad: util.Ptr(true)},
			expected:    Cluster{Local: true, LoadImages: false, PushImages: true},
		},
		{
			description: "docker-for-desktop",
			cfg:         &ContextConfig{Kubecontext: "docker-for-desktop"},
			expected:    Cluster{Local: true, LoadImages: false, PushImages: false},
		},
		{
			description: "minikube",
			cfg:         &ContextConfig{Kubecontext: "minikube"},
			expected:    Cluster{Local: true, LoadImages: false, PushImages: false},
		},
		{
			description: "docker-desktop",
			cfg:         &ContextConfig{Kubecontext: "docker-desktop"},
			expected:    Cluster{Local: true, LoadImages: false, PushImages: false},
		},
		{
			description: "generic cluster with local-cluster=true",
			cfg:         &ContextConfig{Kubecontext: "some-cluster", LocalCluster: util.Ptr(true)},
			expected:    Cluster{Local: true, LoadImages: false, PushImages: false},
		},
		{
			description: "generic cluster with minikube profile",
			cfg:         &ContextConfig{Kubecontext: "some-cluster"},
			profile:     "someprofile",
			expected:    Cluster{Local: true, LoadImages: false, PushImages: false},
		},
		{
			description: "generic cluster",
			cfg:         &ContextConfig{Kubecontext: "anything-else"},
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true},
		},
		{
			description: "not a legacy kind cluster",
			cfg:         &ContextConfig{Kubecontext: "kind@blah"},
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true},
		},
		{
			description: "not a kind cluster",
			cfg:         &ContextConfig{Kubecontext: "other-kind"},
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true},
		},
		{
			description: "not a k3d cluster",
			cfg:         &ContextConfig{Kubecontext: "not-k3d"},
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true},
		},
		{
			description: "generic cluster, default repo already defined",
			cfg:         &ContextConfig{Kubecontext: "anything-else", DefaultRepo: "myrepo"},
			defaultRepo: NewStringOrUndefined(&defaultRepo),
			expected:    Cluster{Local: false, LoadImages: false, PushImages: true, DefaultRepo: NewStringOrUndefined(&defaultRepo)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx := context.Background()

			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, nil })
			t.Override(&DiscoverLocalRegistry, func(context.Context, string) (*string, error) { return test.registry, nil })
			t.Override(&cluster.GetClient, func() cluster.Client { return fakeClient{} })

			cluster, _ := GetCluster(ctx, GetClusterOpts{
				ConfigFile:      "dummyname",
				DefaultRepo:     test.defaultRepo,
				MinikubeProfile: test.profile,
				DetectMinikube:  true,
			})
			t.CheckDeepEqual(test.expected, cluster)

			cluster, _ = GetCluster(ctx, GetClusterOpts{
				ConfigFile:      "dummyname",
				DefaultRepo:     test.defaultRepo,
				MinikubeProfile: test.profile,
				DetectMinikube:  false,
			})
			t.CheckDeepEqual(test.expected, cluster)
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

func TestIsMixedPlatformCluster(t *testing.T) {
	type platform struct {
		os   string
		arch string
	}
	tests := []struct {
		description string
		nodes       []platform
		expected    bool
	}{
		{
			description: "no nodes",
			expected:    false,
		},
		{
			description: "single platform nodes",
			nodes: []platform{
				{os: "linux", arch: "amd64"},
				{os: "linux", arch: "amd64"},
			},
			expected: false,
		},
		{
			description: "mixed platform nodes",
			nodes: []platform{
				{os: "linux", arch: "amd64"},
				{os: "linux", arch: "amd64"},
				{os: "linux", arch: "arm64"},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			nodes := &v1.NodeList{}
			for _, n := range test.nodes {
				nodes.Items = append(nodes.Items, v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: uuid.New().String(),
					},
					Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{MachineID: uuid.New().String(), Architecture: n.arch, OperatingSystem: n.os}},
				})
			}
			cl := fake.NewSimpleClientset(nodes)
			t.Override(&kubeclient.Client, func(kubeContext string) (kubernetes.Interface, error) { return cl, nil })
			t.CheckDeepEqual(test.expected, IsMixedPlatformCluster(context.Background(), ""))
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
			cliValue:     util.Ptr("default/repo"),
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
			cliValue:     util.Ptr(""),
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

func TestGetMultiLevelRepo(t *testing.T) {
	tests := []struct {
		description   string
		cfg           *ContextConfig
		expectedValue *bool
		shouldErr     bool
	}{
		{
			description:   "empty",
			cfg:           &ContextConfig{},
			expectedValue: (*bool)(nil),
		},
		{
			description:   "from global config",
			cfg:           &ContextConfig{MultiLevelRepo: util.Ptr(true)},
			expectedValue: util.Ptr(true),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, nil })

			multiLevelRepo, err := GetMultiLevelRepo("config")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedValue, multiLevelRepo)
		})
	}
}

func TestDiscoverLocalRegistry(t *testing.T) {
	var defaultRepo = "localhost:4000"

	tests := []struct {
		description        string
		registryConfigData map[string]string
		expectedRegistry   *string
		expectedErr        string
	}{
		{
			description: "no config map",
		},
		{
			description:        "missing data",
			registryConfigData: make(map[string]string),
			expectedErr:        "invalid local-registry-hosting ConfigMap",
		},
		{
			description: "invalid data",
			registryConfigData: map[string]string{
				"localRegistryHosting.v1": `invalid`,
			},
			expectedErr: "invalid local-registry-hosting ConfigMap",
		},
		{
			description: "empty data",
			registryConfigData: map[string]string{
				"localRegistryHosting.v1": `{}`,
			},
		},
		{
			description: "valid data",
			registryConfigData: map[string]string{
				"localRegistryHosting.v1": `{host: "localhost:4000"}`,
			},
			expectedRegistry: &defaultRepo,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx := context.Background()
			fakeKubeClient := fake.NewSimpleClientset()

			if test.registryConfigData != nil {
				configMap := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "local-registry-hosting",
					},
					Data: test.registryConfigData,
				}

				_, err := fakeKubeClient.CoreV1().ConfigMaps("kube-public").Create(ctx, configMap, metav1.CreateOptions{})
				t.CheckError(false, err)
			}

			t.Override(&kubeclient.Client, func(kubeContext string) (kubernetes.Interface, error) { return fakeKubeClient, nil })

			registry, err := DiscoverLocalRegistry(ctx, "dummyname")
			t.CheckDeepEqual(test.expectedRegistry, registry)

			if test.expectedErr != "" {
				t.CheckErrorContains(test.expectedErr, err)
			} else {
				t.CheckError(false, err)
			}
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
			err := UpdateHaTSSurveyTaken(cfg)
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

func TestUpdateMsgDisplayed(t *testing.T) {
	testTimeStr := "2021-01-01T00:00:00Z"
	tests := []struct {
		description string
		cfg         string
		expectedCfg *GlobalConfig
	}{
		{
			description: "update global context when context is empty",
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					UpdateCheckConfig: &UpdateConfig{LastPrompted: testTimeStr},
				},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "update global context when update config is not nil",
			cfg: `
global:
  update-config:
    last-prompted: "some date"
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					UpdateCheckConfig: &UpdateConfig{LastPrompted: testTimeStr},
				},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "don't add LastPrompted if UpdateCheck is disabled",
			cfg: `
global:
  update-check: false
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					UpdateCheck: util.Ptr(false),
				},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "don't update LastPrompted if UpdateCheck is disabled",
			cfg: `
global:
  update-check: false
  update:
    last-prompted: "some date"
kubeContexts: []`,
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					UpdateCheck: util.Ptr(false),
					UpdateCheckConfig: &UpdateConfig{
						LastPrompted: "some date"},
				},
				ContextConfigs: []*ContextConfig{},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configFile := t.TempFile("config", []byte(test.cfg))
			t.Override(&ReadConfigFile, ReadConfigFileNoCache)
			t.Override(&current, func() time.Time {
				testTime, _ := time.Parse(time.RFC3339, testTimeStr)
				return testTime
			})

			// update the cfg
			err := UpdateMsgDisplayed(configFile)
			t.CheckNoError(err)

			cfg, cfgErr := ReadConfigFileNoCache(configFile)
			t.CheckErrorAndDeepEqual(false, cfgErr, test.expectedCfg, cfg)
		})
	}
}

func TestShouldDisplayUpdateMsg(t *testing.T) {
	todayStr := time.Now().Format(time.RFC3339)
	yesterday := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
	tests := []struct {
		description string
		cfg         *ContextConfig
		expected    bool
	}{
		{
			description: "should not display prompt when prompt is displayed in last 24 hours",
			cfg: &ContextConfig{
				UpdateCheckConfig: &UpdateConfig{LastPrompted: todayStr},
			},
		},
		{
			description: "should display prompt when last prompted is before 24 hours",
			cfg: &ContextConfig{
				UpdateCheckConfig: &UpdateConfig{LastPrompted: yesterday},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&GetConfigForCurrentKubectx, func(string) (*ContextConfig, error) { return test.cfg, nil })
			t.CheckDeepEqual(test.expected, ShouldDisplayUpdateMsg("dummyconfig"))
		})
	}
}

func TestUpdateUserSurveyTaken(t *testing.T) {
	tests := []struct {
		description string
		cfg         string
		id          string
		expectedCfg *GlobalConfig
	}{
		{
			description: "update global context when user survey is empty",
			id:          "foo",
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					Survey: &SurveyConfig{UserSurveys: []*UserSurvey{
						{ID: "foo", Taken: util.Ptr(true)},
					}}},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "append user survey when not nil",
			cfg: `
global:
  survey:
    user-surveys:
      - id: "foo1"
        taken: true
kubeContexts: []`,
			id: "foo2",
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					Survey: &SurveyConfig{
						UserSurveys: []*UserSurvey{
							{ID: "foo1", Taken: util.Ptr(true)},
							{ID: "foo2", Taken: util.Ptr(true)},
						}}},
				ContextConfigs: []*ContextConfig{},
			},
		},
		{
			description: "update entry for a key in user survey",
			cfg: `
global:
  survey:
    user-surveys:
      - id: "foo"
        taken: false
kubeContexts: []`,
			id: "foo",
			expectedCfg: &GlobalConfig{
				Global: &ContextConfig{
					Survey: &SurveyConfig{
						UserSurveys: []*UserSurvey{
							{ID: "foo", Taken: util.Ptr(true)},
						}}},
				ContextConfigs: []*ContextConfig{},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := t.TempFile("config", []byte(test.cfg))
			t.Override(&ReadConfigFile, ReadConfigFileNoCache)

			// update the time
			err := UpdateUserSurveyTaken(cfg, test.id)
			t.CheckNoError(err)

			actualConfig, cfgErr := ReadConfigFile(cfg)
			t.CheckNoError(cfgErr)
			t.CheckDeepEqual(test.expectedCfg, actualConfig)
		})
	}
}
