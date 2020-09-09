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
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	kubeContext = "test-kubeContext"
	namespace   = "test-namespace"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		description     string
		shouldErr       bool
		cfg             Config
		expectedBuilder *Builder
	}{
		{
			description: "failed to parse cluster build timeout",
			cfg: &mockConfig{
				cluster: latest.ClusterDetails{
					Timeout: "illegal",
				},
			},
			shouldErr: true,
		},
		{
			description: "cluster builder inherits the config",
			cfg: &mockConfig{
				kubeContext: kubeContext,
				namespace:   namespace,
				cluster: latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
			},
			shouldErr: false,
			expectedBuilder: &Builder{
				ClusterDetails: &latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
				timeout:            100 * time.Second,
				insecureRegistries: nil,
				kubeContext:        kubeContext,
				kubectlcli: &kubectl.CLI{
					KubeContext: kubeContext,
					Namespace:   namespace,
				},
				muted: config.Muted{},
			},
		},
		{
			description: "insecure registries are taken from the run context",
			cfg: &mockConfig{
				kubeContext:        kubeContext,
				namespace:          namespace,
				insecureRegistries: map[string]bool{"insecure-reg1": true},
				cluster: latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
			},
			shouldErr: false,
			expectedBuilder: &Builder{
				ClusterDetails: &latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
				timeout:            100 * time.Second,
				insecureRegistries: map[string]bool{"insecure-reg1": true},
				kubeContext:        kubeContext,
				kubectlcli: &kubectl.CLI{
					KubeContext: kubeContext,
					Namespace:   namespace,
				},
				muted: config.Muted{},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			builder, err := NewBuilder(test.cfg)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expectedBuilder, builder, cmp.AllowUnexported(Builder{}, kubectl.CLI{}, sync.Once{}, sync.Mutex{}))
			}
		})
	}
}

func TestPruneIsNoop(t *testing.T) {
	pruneError := (&Builder{}).Prune(context.TODO(), nil)
	testutil.CheckDeepEqual(t, nil, pruneError)
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	kubeContext           string
	namespace             string
	insecureRegistries    map[string]bool
	cluster               latest.ClusterDetails
}

func (c *mockConfig) GetKubeContext() string                 { return c.kubeContext }
func (c *mockConfig) GetKubeNamespace() string               { return c.namespace }
func (c *mockConfig) GetInsecureRegistries() map[string]bool { return c.insecureRegistries }
func (c *mockConfig) Pipeline() latest.Pipeline {
	var pipeline latest.Pipeline
	pipeline.Build.BuildType.Cluster = &c.cluster
	return pipeline
}
