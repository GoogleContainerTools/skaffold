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

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		description string
		dir         string
		filename    string
		args        []string
		deployments []string
		pods        []string
		env         []string
		remoteOnly  bool
	}{
		{
			description: "getting-started",
			dir:         "examples/getting-started",
			pods:        []string{"getting-started"},
		}, {
			description: "nodejs",
			dir:         "examples/nodejs",
			pods:        []string{"node"},
		}, {
			description: "structure-tests",
			dir:         "examples/structure-tests",
			pods:        []string{"getting-started"},
		}, {
			description: "microservices",
			dir:         "examples/microservices",
			deployments: []string{"leeroy-app", "leeroy-web"},
		}, {
			description: "envTagger",
			dir:         "examples/tagging-with-environment-variables",
			pods:        []string{"getting-started"},
			env:         []string{"FOO=foo"},
		}, {
			description: "bazel",
			dir:         "examples/bazel",
			pods:        []string{"bazel"},
		}, {
			description: "Google Cloud Build",
			dir:         "examples/structure-tests",
			args:        []string{"-p", "gcb"},
			pods:        []string{"getting-started"},
			remoteOnly:  true,
		}, {
			description: "Google Cloud Build - sub folder",
			dir:         "testdata/gcb-sub-folder",
			pods:        []string{"getting-started"},
			remoteOnly:  true,
		}, {
			description: "kaniko",
			dir:         "examples/kaniko",
			pods:        []string{"getting-started-kaniko"},
			remoteOnly:  true,
		}, {
			description: "kaniko local",
			dir:         "examples/kaniko-local",
			pods:        []string{"getting-started-kaniko"},
			remoteOnly:  true,
		}, {
			description: "kaniko local - sub folder",
			dir:         "testdata/kaniko-sub-folder",
			pods:        []string{"getting-started-kaniko"},
			remoteOnly:  true,
		}, {
			description: "helm",
			dir:         "examples/helm-deployment",
			deployments: []string{"skaffold-helm"},
			remoteOnly:  true,
		}, {
			description: "docker plugin in gcb exec environment",
			dir:         "testdata/plugin/gcb",
			deployments: []string{"leeroy-app", "leeroy-web"},
		}, {
			description: "bazel plugin in local exec environment",
			dir:         "testdata/plugin/local/bazel",
			pods:        []string{"bazel"},
		}, {
			description: "docker plugin in local exec environment",
			dir:         "testdata/plugin/local/docker",
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.remoteOnly && os.Getenv("REMOTE_INTEGRATION") != "true" {
				t.Skip("skipping remote only test")
			}

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			RunSkaffold(t, "run", test.dir, ns.Name, test.filename, test.env)

			for _, p := range test.pods {
				if err := kubernetesutil.WaitForPodReady(context.Background(), client.CoreV1().Pods(ns.Name), p); err != nil {
					t.Fatalf("Timed out waiting for pod ready")
				}
			}

			for _, d := range test.deployments {
				if err := kubernetesutil.WaitForDeploymentToStabilize(context.Background(), client, ns.Name, d, 10*time.Minute); err != nil {
					t.Fatalf("Timed out waiting for deployment to stabilize")
				}
			}

			RunSkaffold(t, "delete", test.dir, ns.Name, test.filename, test.env)
		})
	}
}
