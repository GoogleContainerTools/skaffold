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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
		cluster     *latestV1.ClusterDetails
	}{
		{
			description: "failed to parse cluster build timeout",
			bCtx:        &mockBuilderContext{},
			cluster: &latestV1.ClusterDetails{
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
			cluster: &latestV1.ClusterDetails{
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
			cluster: &latestV1.ClusterDetails{
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
