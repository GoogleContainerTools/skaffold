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
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	kubeContext = "test-kubeContext"
	namespace   = "test-namespace"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		bCtx        BuilderContext
		cluster     *latest.ClusterDetails
	}{
		{
			description: "failed to parse cluster build timeout",
			bCtx:        &mockBuilderContext{},
			cluster: &latest.ClusterDetails{
				Timeout: "illegal",
			},
			shouldErr: true,
		},
		{
			description: "cluster builder inherits the config",
			bCtx: &mockBuilderContext{
				kubeContext: kubeContext,
				namespace:   namespace,
			},
			cluster: &latest.ClusterDetails{
				Timeout:   "100s",
				Namespace: "test-ns",
			},
		},
		{
			description: "insecure registries are taken from the run context",
			bCtx: &mockBuilderContext{
				kubeContext:        kubeContext,
				namespace:          namespace,
				insecureRegistries: map[string]bool{"insecure-reg1": true},
			},
			cluster: &latest.ClusterDetails{
				Timeout:   "100s",
				Namespace: "test-ns",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewBuilder(test.bCtx, test.cluster)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestPruneIsNoop(t *testing.T) {
	pruneError := (&Builder{}).Prune(context.TODO(), nil)
	testutil.CheckDeepEqual(t, nil, pruneError)
}

type mockBuilderContext struct {
	runcontext.RunContext // Embedded to provide the default values.
	kubeContext           string
	namespace             string
	insecureRegistries    map[string]bool
	runMode               config.RunMode
	artifactStore         build.ArtifactStore
}

func (c *mockBuilderContext) GetKubeContext() string                 { return c.kubeContext }
func (c *mockBuilderContext) GetKubeNamespace() string               { return c.namespace }
func (c *mockBuilderContext) GetInsecureRegistries() map[string]bool { return c.insecureRegistries }
func (c *mockBuilderContext) Mode() config.RunMode                   { return c.runMode }
func (c *mockBuilderContext) ArtifactStore() build.ArtifactStore     { return c.artifactStore }

type namespaceBuilderContext struct {
	runcontext.RunContext
}

func (c *namespaceBuilderContext) ArtifactStore() build.ArtifactStore { return nil }

func TestNewBuilderNamespace(t *testing.T) {
	for _, tc := range []struct {
		name            string
		context         string
		optionNamespace string
		buildNamespace  string
		deployNamespace string
		explicitConfig  bool
		expected        string
	}{
		{name: "current context namespace", expected: "current-ns"},
		{name: "selected context namespace", context: "selected", expected: "selected-ns"},
		{name: "deployment namespace does not override build context", context: "selected", deployNamespace: "deploy-ns", expected: "selected-ns"},
		{name: "deployment namespace does not override default build namespace", context: "empty", deployNamespace: "deploy-ns", expected: "default"},
		{name: "selected explicit kubeconfig", context: "selected", explicitConfig: true, expected: "selected-ns"},
		{name: "CLI or environment namespace", context: "selected", optionNamespace: "env-ns", expected: "env-ns"},
		{name: "option namespace without kube context", context: "missing", optionNamespace: "env-ns", expected: "env-ns"},
		{name: "explicit build namespace", context: "selected", optionNamespace: "env-ns", buildNamespace: "build-ns", expected: "build-ns"},
		{name: "unset context namespace", context: "empty", expected: "default"},
		{name: "explicit default namespace", context: "default", expected: "default"},
	} {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			kubeConfig := t.TempFile("config", []byte(`apiVersion: v1
kind: Config
current-context: current
contexts:
- name: current
  context:
    namespace: current-ns
- name: selected
  context:
    namespace: selected-ns
- name: empty
  context: {}
- name: default
  context:
    namespace: default
`))
			opts := config.SkaffoldOptions{Namespace: tc.optionNamespace}
			if tc.explicitConfig {
				opts.KubeConfig = kubeConfig
				t.Setenv("KUBECONFIG", t.TempFile("other-config", []byte("invalid")))
			} else {
				t.Setenv("KUBECONFIG", kubeConfig)
			}
			cfg := &latest.SkaffoldConfig{Pipeline: latest.Pipeline{Build: latest.BuildConfig{
				BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{Namespace: tc.buildNamespace}},
			}}}
			t.RequireNoError(defaults.Set(cfg))
			t.CheckDeepEqual(tc.buildNamespace, cfg.Build.Cluster.Namespace)
			pipeline := cfg.Pipeline
			if tc.deployNamespace != "" {
				pipeline.Deploy.KubectlDeploy = &latest.KubectlDeploy{DefaultNamespace: &tc.deployNamespace}
			}
			builder, err := NewBuilder(&namespaceBuilderContext{RunContext: runcontext.RunContext{
				KubeContext: tc.context, Opts: opts,
				Pipelines: runcontext.NewPipelines(map[string]latest.Pipeline{"test": pipeline}, []string{"test"}),
			}}, cfg.Build.Cluster)
			t.RequireNoError(err)
			t.CheckDeepEqual(tc.expected, builder.Namespace)
			// Do not bake context-dependent defaults into a reusable pipeline.
			t.CheckDeepEqual(tc.buildNamespace, cfg.Build.Cluster.Namespace)
		})
	}
}
