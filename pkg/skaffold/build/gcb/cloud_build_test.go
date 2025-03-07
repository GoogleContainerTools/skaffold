/*
Copyright 2025 The Skaffold Authors

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

package gcb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
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
