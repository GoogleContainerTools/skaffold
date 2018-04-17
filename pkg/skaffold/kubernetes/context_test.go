/*
Copyright 2018 Google LLC

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

package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCurrentContext(t *testing.T) {
	tmpDir := os.TempDir()
	kubeConfig := filepath.Join(tmpDir, "config")
	defer os.Remove(kubeConfig)
	if err := clientcmd.WriteToFile(api.Config{
		CurrentContext: "cluster1",
	}, kubeConfig); err != nil {
		t.Fatalf("writing temp kubeconfig")
	}
	unsetEnvs := testutil.SetEnvs(t, map[string]string{"KUBECONFIG": kubeConfig})
	defer unsetEnvs(t)
	context, err := CurrentContext()
	testutil.CheckErrorAndDeepEqual(t, false, err, "cluster1", context)
}
