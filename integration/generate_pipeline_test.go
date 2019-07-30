package integration

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestGeneratePipeline(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		args        []string
	}{
		{
			description: "no profiles",
			dir:         "testdata/generate_pipeline",
		},
		{
			description: "existing oncluster profile",
			dir:         "testdata/generate_pipeline",
		},
		{
			description: "existing other profile",
			dir:         "testdata/generate_pipeline",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			skaffold.GeneratePipeline().InDir(test.dir).RunOrFail(t)
		})
	}
}
