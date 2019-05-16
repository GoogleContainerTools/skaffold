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

package cluster

import (
	"testing"

	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRetrieveEnv(t *testing.T) {
	builder, err := NewBuilder(&runcontext.RunContext{
		KubeContext: "kubecontext",
		Cfg: &latest.Pipeline{
			Build: latest.BuildConfig{
				BuildType: latest.BuildType{
					Cluster: &latest.ClusterDetails{
						Namespace:      "namespace",
						PullSecretName: "pullSecret",
						DockerConfig: &latest.DockerConfig{
							SecretName: "dockerconfig",
						},
						Timeout: "2m",
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("err retrieving builder: %v", err)
	}

	actual := builder.retrieveExtraEnv()
	expected := []string{"KUBE_CONTEXT=kubecontext", "NAMESPACE=namespace", "PULL_SECRET_NAME=pullSecret", "DOCKER_CONFIG_SECRET_NAME=dockerconfig", "TIMEOUT=2m"}
	testutil.CheckErrorAndDeepEqual(t, false, nil, expected, actual)
}
