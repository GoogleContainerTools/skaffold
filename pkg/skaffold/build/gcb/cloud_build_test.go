package gcb

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/stretchr/testify/assert"
)

func TestRegionFromWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		workerPool string
		expected   string
	}{
		{
			name:       "valid worker pool",
			workerPool: "projects/my-project/locations/us-central1/workerPools/my-pool",
			expected:   "us-central1",
		},
		{
			name:       "valid worker pool with different region",
			workerPool: "projects/my-project/locations/eu-west1/workerPools/another-pool",
			expected:   "eu-west1",
		},
		// Not testing invalid worker pool because format is already tested in the validation tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				GoogleCloudBuild: &latest.GoogleCloudBuild{
					WorkerPool: tt.workerPool,
				},
			}
			region := b.regionFromWorkerPool()
			assert.Equal(t, tt.expected, region)
		})
	}
}
