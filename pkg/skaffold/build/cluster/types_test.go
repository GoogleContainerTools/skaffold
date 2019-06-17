package cluster

import (
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestNewBuilder(t *testing.T) {

	tcs := []struct {
		name            string
		shouldErr       bool
		clusterDetails  *latest.ClusterDetails
		expectedBuilder *Builder
	}{
		{
			name: "failed to parse cluster build timeout",
			clusterDetails: &latest.ClusterDetails{
				Timeout: "illegal",
			},
			shouldErr: true,
		},
		{
			name: "cluster builder inherits the config",
			clusterDetails: &latest.ClusterDetails{
				Timeout:   "100s",
				Namespace: "test-ns",
			},
			shouldErr: false,
			expectedBuilder: &Builder{
				ClusterDetails: &latest.ClusterDetails{
					Timeout:   "100s",
					Namespace: "test-ns",
				},
				timeout:            100 * time.Second,
				insecureRegistries: nil,
			},
		},
	}
	for _, tc := range tcs {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			builder, err := NewBuilder(stubRunContext(tc.clusterDetails))
			t.CheckError(tc.shouldErr, err)
			if !tc.shouldErr {
				t.CheckDeepEqual(tc.expectedBuilder, builder, cmp.AllowUnexported(Builder{}))
			}
		})
	}
}

func TestLabels(t *testing.T) {
	testutil.CheckDeepEqual(t, map[string]string{"skaffold.dev/builder": "cluster"}, (&Builder{}).Labels())
}

func TestPruneIsNoop(t *testing.T) {
	testutil.CheckDeepEqual(t, nil, (&Builder{}).Prune(nil, nil))
}

func TestSyncMapNotSupported(t *testing.T) {
	syncMap, err := (&Builder{}).SyncMap(nil, nil)
	var expected map[string][]string
	testutil.CheckErrorAndDeepEqual(t, true, err, expected, syncMap)
}

func stubRunContext(clusterDetails *latest.ClusterDetails) *context.RunContext {
	return &context.RunContext{
		Cfg: &latest.Pipeline{
			Build: latest.BuildConfig{
				BuildType: latest.BuildType{
					Cluster: clusterDetails,
				},
			},
		},
		Opts: &config.SkaffoldOptions{
			NoPrune:        false,
			CacheArtifacts: false,
			SkipTests:      false,
		},
	}
}
