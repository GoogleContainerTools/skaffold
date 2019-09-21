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
    user: user1
  name: cluster-bar
current-context: cluster-foo
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
		logrus.SetLevel(logrus.DebugLevel)
		t.SetEnvs(map[string]string{"KUBECONFIG": "non-valid"})
		resetConfig()

		_, err := getRestClientConfig("")

		if err == nil {
			t.Errorf("expected error outside the cluster")
		}
	})
}

func TestUseKubeContext(t *testing.T) {
	tests := []struct {
		name              string
		cliValue          string
		firstYamlValue    string
		secondYamlValue   string
		lockWithFirstCall bool
		expected          string
	}{
		{
			name:            "cliValue takes preference",
			cliValue:        "initial",
			secondYamlValue: "other",
			expected:        "initial",
		},
		{
			name:            "set kube-context with second call",
			secondYamlValue: "other",
			expected:        "other",
		},
		{
			name:            "override kube-context with second call",
			firstYamlValue:  "initial",
			secondYamlValue: "other",
			expected:        "other",
		},
		{
			name:              "cliValue takes preference before locking",
			cliValue:          "initial",
			firstYamlValue:    "other",
			expected:          "initial",
			lockWithFirstCall: true,
		},
		{
			name:              "cliValue takes preference after locking",
			cliValue:          "initial",
			secondYamlValue:   "other",
			expected:          "initial",
			lockWithFirstCall: true,
		},
		{
			name:              "cannot overwrite kube-context after it is locked",
			firstYamlValue:    "initial",
			secondYamlValue:   "other",
			expected:          "initial",
			lockWithFirstCall: true,
		},
		{
			name:              "cannot set kube-context after it is locked",
			secondYamlValue:   "other",
			lockWithFirstCall: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			UseKubeContext(test.cliValue, test.firstYamlValue, test.lockWithFirstCall)
			UseKubeContext(test.cliValue, test.secondYamlValue, false)

			t.CheckDeepEqual(test.expected, kubeContext)
			isKubeContextLocked = false // cleanup
		})
	}
}

func resetKubeConfig(t *testutil.T, content string) {
	kubeConfig := t.TempFile("config", []byte(content))
	kubeContext = ""
	isKubeContextLocked = false
	t.SetEnvs(map[string]string{"KUBECONFIG": kubeConfig})
	resetConfig()
}
