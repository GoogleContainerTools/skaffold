package runner

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/examples/bazel/bazel-bazel/external/go_sdk/src/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

//TODO(marlon-gamez): beef up testing
func TestGeneratePipeline(t *testing.T) {
	var tests = []struct {
		description    string
		testBench      *TestBench
		skaffoldConfig *latest.SkaffoldConfig
		shouldErr      bool
	}{
		{
			description: "successful pipeline generation",
			testBench:   &TestBench{},
			skaffoldConfig: &latest.SkaffoldConfig{
				APIVersion: "skaffold/v1beta11",
				Kind:       "Config",
			},
			shouldErr: false,
		},
		{
			description:    "failed pipeline generation",
			testBench:      &TestBench{},
			skaffoldConfig: &latest.SkaffoldConfig{},
			shouldErr:      true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {

			// tempDir for writing file to
			tempDir := t.NewTempDir()
			filePath := tempDir.Root() + "/pipeline.yaml"

			runner := createRunner(t, test.testBench, nil)
			out := new(bytes.Buffer)
			err := runner.GeneratePipeline(context.Background(), out, test.skaffoldConfig, filePath)

			t.CheckError(test.shouldErr, err)
		})
	}
}
