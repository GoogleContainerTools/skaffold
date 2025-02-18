package gcb

import (
	"testing"
)

func TestGetRegionFromWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		workerPool string
		want       string
		wantErr    bool
	}{
		{
			name:       "valid worker pool",
			workerPool: "projects/my-project/locations/us-central1/workerPools/my-pool",
			want:       "us-central1",
			wantErr:    false,
		},
		{
			name:       "invalid worker pool when one is expected",
			workerPool: "",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRegionFromWorkerPool(tt.workerPool)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRegionFromWorkerPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getRegionFromWorkerPool() = %v, want %v", got, tt.want)
			}
		})
	}
}


