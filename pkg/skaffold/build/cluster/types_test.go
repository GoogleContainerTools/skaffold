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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	kubeContext = "test-kubeContext"
	namespace   = "test-namespace"
)

var (
	cfg = &clusterConfig{
		kubeContext: kubeContext,
		namespace:   namespace,
		cluster: latest.ClusterDetails{
			Timeout:   "100s",
			Namespace: "test-ns",
		},
	}

	cfgWithInsecureRegistries = &clusterConfig{
		kubeContext:        kubeContext,
		namespace:          namespace,
		insecureRegistries: map[string]bool{"insecure-reg1": true},
		cluster: latest.ClusterDetails{
			Timeout:   "100s",
			Namespace: "test-ns",
		},
	}

	cfgWithInvalidTimeout = &clusterConfig{
		kubeContext: kubeContext,
		namespace:   namespace,
		cluster: latest.ClusterDetails{
			Timeout: "illegal",
		},
	}
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		cfg         Config
		expected    *Builder
	}{
		{
			description: "failed to parse cluster build timeout",
			cfg:         cfgWithInvalidTimeout,
			shouldErr:   true,
		},
		{
			description: "cluster builder inherits the config",
			cfg:         cfg,
			expected: &Builder{
				ClusterDetails: &latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
				cfg:        cfg,
				kubectlcli: kubectl.NewCLI(cfg),
				timeout:    100 * time.Second,
			},
		},
		{
			description: "insecure registries",
			cfg:         cfgWithInsecureRegistries,
			expected: &Builder{
				ClusterDetails: &latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
				cfg:        cfgWithInsecureRegistries,
				kubectlcli: kubectl.NewCLI(cfgWithInsecureRegistries),
				timeout:    100 * time.Second,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			builder, err := NewBuilder(test.cfg)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expected, builder, cmp.AllowUnexported(Builder{}, sync.Once{}, sync.Mutex{}, clusterConfig{}, kubectl.CLI{}))
			}
		})
	}
}

func TestPruneIsNoop(t *testing.T) {
	pruneError := (&Builder{}).Prune(context.TODO(), nil)
	testutil.CheckDeepEqual(t, nil, pruneError)
}

type clusterConfig struct {
	Config

	kubeContext        string
	namespace          string
	insecureRegistries map[string]bool
	cluster            latest.ClusterDetails
}

func (c *clusterConfig) GetKubeContext() string              { return c.kubeContext }
func (c *clusterConfig) GetKubeNamespace() string            { return c.namespace }
func (c *clusterConfig) InsecureRegistries() map[string]bool { return c.insecureRegistries }
func (c *clusterConfig) Pipeline() latest.Pipeline {
	var pipeline latest.Pipeline
	pipeline.Build.BuildType.Cluster = &c.cluster
	return pipeline
}
