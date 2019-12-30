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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	kubeContext = "test-kubeContext"
	namespace   = "test-namespace"
)

func TestNewBuilderFail(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		_, err := NewBuilder(stubRunContext(&latest.ClusterDetails{
			Timeout: "illegal",
		}, nil), nil)

		t.CheckError(true, err)
	})
}

func TestLabels(t *testing.T) {
	testutil.CheckDeepEqual(t, map[string]string{"skaffold.dev/builder": "cluster"}, (&Builder{}).Labels())
}

func TestPruneIsNoop(t *testing.T) {
	pruneError := (&Builder{}).Prune(context.TODO(), nil)
	testutil.CheckDeepEqual(t, nil, pruneError)
}

func TestSyncMapNotSupported(t *testing.T) {
	syncMap, err := (&Builder{}).SyncMap(context.TODO(), nil)
	var expected map[string][]string
	testutil.CheckErrorAndDeepEqual(t, true, err, expected, syncMap)
}

func stubRunContext(clusterDetails *latest.ClusterDetails, insecureRegistries map[string]bool) *runcontext.RunContext {
	pipeline := latest.Pipeline{}
	pipeline.Build.BuildType.Cluster = clusterDetails

	return &runcontext.RunContext{
		Cfg:                pipeline,
		InsecureRegistries: insecureRegistries,
		KubeContext:        kubeContext,
		Opts: config.SkaffoldOptions{
			Namespace: namespace,
		},
	}
}
