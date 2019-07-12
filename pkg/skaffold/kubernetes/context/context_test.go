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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

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

func TestCurrentContext(t *testing.T) {
	testutil.Run(t, "valid context", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		config, err := CurrentConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("cluster-foo", config.CurrentContext)
	})

	testutil.Run(t, "valid with override context", func(t *testutil.T) {
		resetKubeConfig(t, validKubeConfig)

		kubeContext = "cluster-bar"
		config, err := CurrentConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("cluster-bar", config.CurrentContext)
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

		kubeContext = "cluster-bar"
		cfg, err := GetRestClientConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("https://bar.com", cfg.Host)
	})

	testutil.Run(t, "invalid context", func(t *testutil.T) {
		resetKubeConfig(t, "invalid")

		_, err := GetRestClientConfig()

		t.CheckError(true, err)
	})
}

func resetKubeConfig(t *testutil.T, content string) {
	kubeConfig := t.TempFile("config", []byte(content))
	kubeContext = ""
	t.SetEnvs(map[string]string{"KUBECONFIG": kubeConfig})
	resetConfig()
}
