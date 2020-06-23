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

package context

import (
	"io/ioutil"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

const clusterFooContext = "cluster-foo"
const clusterBarContext = "cluster-bar"

const validKubeConfig = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://foo.com
  name: cluster-foo
- cluster:
    server: https://bar.com
  name: cluster-bar
contexts:
- context:
    cluster: cluster-foo
    user: user1
  name: cluster-foo
- context:
    cluster: cluster-bar
    user: user1
  name: cluster-bar
current-context: cluster-foo
users:
- name: user1
  user:
    password: secret
    username: user
`

const changedKubeConfig = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://changed-url.com
  name: cluster-bar
contexts:
- context:
    cluster: cluster-bar
    user: user-bar
  name: context-bar
- context:
    cluster: cluster-bar
    user: user-baz
  name: context-baz
current-context: context-baz
users:
- name: user1
  user:
    password: secret
    username: user
`

func TestCurrentContext(t *testing.T) {
	testutil.Run(t, "valid context", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		config, err := CurrentConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual(clusterFooContext, config.CurrentContext)
	})

	testutil.Run(t, "valid with override context", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		kubeContext = "cluster-bar"
		config, err := CurrentConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual(clusterBarContext, config.CurrentContext)
	})

	testutil.Run(t, "kubeconfig CLI flag takes precedence", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)
		kubeConfig := t.TempFile("config", []byte(changedKubeConfig))

		kubeConfigFile = kubeConfig
		config, err := CurrentConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("context-baz", config.CurrentContext)
	})

	testutil.Run(t, "invalid context", func(t *testutil.T) {
		resetKubeConfig(t, "invalid")

		_, err := CurrentConfig()

		t.CheckError(true, err)
	})
}

func TestGetRestClientConfig(t *testing.T) {
	testutil.Run(t, "valid context", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		cfg, err := GetRestClientConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("https://foo.com", cfg.Host)
	})

	testutil.Run(t, "valid context with override", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		kubeContext = clusterBarContext
		cfg, err := GetRestClientConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("https://bar.com", cfg.Host)
	})

	testutil.Run(t, "invalid context", func(t *testutil.T) {
		resetKubeConfig(t, "invalid")

		_, err := GetRestClientConfig()

		t.CheckError(true, err)
	})

	testutil.Run(t, "kube-config immutability", func(t *testutil.T) {
		logrus.SetLevel(logrus.InfoLevel)
		kubeConfig := t.TempFile("config", []byte(validKubeConfig))
		kubeContext = clusterBarContext
		t.SetEnvs(map[string]string{"KUBECONFIG": kubeConfig})
		resetConfig()

		cfg, err := GetRestClientConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("https://bar.com", cfg.Host)

		if err = ioutil.WriteFile(kubeConfig, []byte(changedKubeConfig), 0644); err != nil {
			t.Error(err)
		}

		cfg, err = GetRestClientConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("https://bar.com", cfg.Host)
	})

	testutil.Run(t, "change context after first execution", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		_, err := GetRestClientConfig()
		t.CheckNoError(err)
		kubeContext = clusterBarContext
		cfg, err := GetRestClientConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("https://bar.com", cfg.Host)
	})

	testutil.Run(t, "REST client in-cluster", func(t *testutil.T) {
		t.SetEnvs(map[string]string{"KUBECONFIG": "non-valid"})
		resetConfig()

		_, err := getRestClientConfig("", "")

		if err == nil {
			t.Errorf("expected error outside the cluster")
		}
	})
}

func TestUseKubeContext(t *testing.T) {
	type invocation struct {
		cliValue, yamlValue string
	}
	tests := []struct {
		name        string
		invocations []invocation
		expected    string
	}{
		{
			name:        "when not called at all",
			invocations: nil,
			expected:    "",
		},
		{
			name:        "yaml value when no CLI value is given",
			invocations: []invocation{{yamlValue: "context2"}},
			expected:    "context2",
		},
		{
			name: "yaml value when no CLI value is given, first invocation persists",
			invocations: []invocation{
				{yamlValue: "context-first"},
				{yamlValue: "context-second"},
			},
			expected: "context-first",
		},
		{
			name:        "CLI value takes precedence",
			invocations: []invocation{{cliValue: "context1", yamlValue: "context2"}},
			expected:    "context1",
		},
		{
			name: "first CLI value takes precedence",
			invocations: []invocation{
				{cliValue: "context-first"},
				{cliValue: "context-second"},
			},
			expected: "context-first",
		},
		{
			name: "mixed CLI value and yaml value - I",
			invocations: []invocation{
				{cliValue: "context-first"},
				{yamlValue: "context-second"},
			},
			expected: "context-first",
		},
		{
			name: "mixed CLI value and yaml value - II",
			invocations: []invocation{
				{yamlValue: "context-first"},
				{cliValue: "context-second"},
			},
			expected: "context-first",
		},
		{
			name: "mixed CLI value and yaml value - III",
			invocations: []invocation{
				{yamlValue: "context-first"},
				{cliValue: "context-second", yamlValue: "context-third"},
			},
			expected: "context-first",
		},
		{
			name: "mixed CLI value and yaml value - IV",
			invocations: []invocation{
				{cliValue: "context-first", yamlValue: "context-second"},
				{cliValue: "context-third", yamlValue: "context-fourth"},
			},
			expected: "context-first",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			kubeContext = ""
			for _, inv := range test.invocations {
				ConfigureKubeConfig("", inv.cliValue, inv.yamlValue)
			}

			t.CheckDeepEqual(test.expected, kubeContext)
			resetConfig()
		})
	}
}

// resetConfig is used by tests
func resetConfig() {
	kubeConfigOnce = sync.Once{}
	configureOnce = sync.Once{}
}

func resetKubeConfig(t *testutil.T, content string) {
	kubeConfig := t.TempFile("config", []byte(content))
	t.SetEnvs(map[string]string{"KUBECONFIG": kubeConfig})
	kubeContext = ""
	kubeConfigFile = ""
	resetConfig()
}
