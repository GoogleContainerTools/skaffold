/*
Copyright 2021 The Skaffold Authors

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

package cmd

import (
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/cluster"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPreReleaseVersion(t *testing.T) {
	tests := []struct {
		description string
		versionStr  string
		expected    bool
	}{
		{
			description: "pre release version",
			versionStr:  "v1.20.0-42-g92900f245-dirty",
			expected:    true,
		},
		{
			description: "release version",
			versionStr:  "v1.20.0",
		},
		{
			description: "incorrect version indicates pre release or dev version",
			versionStr:  "blah",
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := preReleaseVersion(test.versionStr)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestUpdateCheck(t *testing.T) {
	tests := []struct {
		description string
		versionStr  string
		expected    string
		versionMsg  string
		updateCheck bool
	}{
		{
			description: "pre release version, update check is disabled",
			versionStr:  "v1.20.0-42-g92900f245-dirty",
		},
		{
			description: "release version but update check disabled",
			versionStr:  "v1.20.0",
		},
		{
			description: "release version and update check enabled",
			versionStr:  "v1.20.0",
			updateCheck: true,
			versionMsg:  "newer version is available",
			expected:    "newer version is available",
		},
		{
			description: "update check enabled but version already up to date",
			versionStr:  "v1.20.0",
			updateCheck: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&updateCheck, func(string) (string, error) {
				return test.versionMsg, nil
			})
			t.Override(&update.EnableCheck, test.updateCheck)
			actual := updateCheckForReleasedVersionsIfNotDisabled(test.versionStr)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

type envNamespaceBuilderContext struct {
	runcontext.RunContext
}

func (c *envNamespaceBuilderContext) ArtifactStore() build.ArtifactStore { return nil }

func TestClusterNamespaceFromEnvFile(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      string
		expected string
	}{
		{name: "namespace from env file", env: "SKAFFOLD_NAMESPACE=env-ns\n", expected: "env-ns"},
		{name: "selected context from env file", expected: "selected-ns"},
	} {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			t.Override(&opts, config.SkaffoldOptions{})
			t.UnsetEnv("SKAFFOLD_NAMESPACE")
			t.UnsetEnv("SKAFFOLD_DEPLOY_NAMESPACE")
			t.UnsetEnv("SKAFFOLD_KUBE_CONTEXT")
			t.UnsetEnv("SKAFFOLD_KUBECONFIG")
			t.Setenv("KUBECONFIG", t.TempFile("other-config", []byte("invalid")))
			tmp := t.NewTempDir()
			tmp.Write("config", `apiVersion: v1
kind: Config
current-context: current
contexts:
- name: current
  context:
    namespace: current-ns
- name: selected
  context:
    namespace: selected-ns
`)
			tmp.Write("skaffold.env", tc.env+"SKAFFOLD_KUBE_CONTEXT=selected\nSKAFFOLD_KUBECONFIG=config\n")
			tmp.Chdir()
			NewSkaffoldCommand(io.Discard, io.Discard)
			t.CheckDeepEqual("selected", opts.KubeContext)
			t.CheckDeepEqual("config", opts.KubeConfig)
			cfg := &latest.SkaffoldConfig{Pipeline: latest.Pipeline{Build: latest.BuildConfig{
				BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}},
			}}}
			t.RequireNoError(defaults.Set(cfg))
			builder, err := cluster.NewBuilder(&envNamespaceBuilderContext{RunContext: runcontext.RunContext{
				KubeContext: opts.KubeContext, Opts: opts,
			}}, cfg.Build.Cluster)
			t.RequireNoError(err)
			t.CheckDeepEqual(tc.expected, builder.Namespace)
		})
	}
}
