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

func TestCurrentContext(t *testing.T) {
	testutil.Run(t, "valid context", func(t *testutil.T) {
		resetKubeConfig(t, "apiVersion: v1\nkind: Config\ncurrent-context: cluster1\n")

		config, err := CurrentConfig()

		t.CheckNoError(err)
		t.CheckDeepEqual("cluster1", config.CurrentContext)
	})

	testutil.Run(t, "invalid context", func(t *testutil.T) {
		resetKubeConfig(t, "invalid")

		_, err := CurrentConfig()

		t.CheckError(true, err)
	})
}

func resetKubeConfig(t *testutil.T, content string) {
	kubeConfig := t.TempFile("config", []byte(content))
	t.SetEnvs(map[string]string{"KUBECONFIG": kubeConfig})
	ResetCurrentConfig()
}
