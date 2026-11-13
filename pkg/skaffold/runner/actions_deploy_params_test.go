/*
Copyright 2026 The Skaffold Authors

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

package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestMergeDeployParams(t *testing.T) {
	tests := []struct {
		description string
		base        map[string]string
		fileContent string // empty means no value file
		overrides   []string
		expected    map[string]string
	}{
		{
			description: "no inputs yields nil",
			expected:    nil,
		},
		{
			description: "base-only is preserved",
			base:        map[string]string{"A": "1"},
			expected:    map[string]string{"A": "1"},
		},
		{
			description: "overrides only",
			overrides:   []string{"FOO=bar", "BAZ=qux"},
			expected:    map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			description: "value-file only",
			fileContent: "KEY=val\nOTHER=thing\n",
			expected:    map[string]string{"KEY": "val", "OTHER": "thing"},
		},
		{
			description: "overrides shadow value-file which shadow base",
			base:        map[string]string{"COMMON": "from-base", "ONLY_BASE": "b"},
			fileContent: "COMMON=from-file\nONLY_FILE=f\n",
			overrides:   []string{"COMMON=from-override", "ONLY_OVERRIDE=o"},
			expected: map[string]string{
				"COMMON":        "from-override",
				"ONLY_BASE":     "b",
				"ONLY_FILE":     "f",
				"ONLY_OVERRIDE": "o",
			},
		},
		{
			description: "override without = sign is skipped",
			overrides:   []string{"FOO=bar", "MALFORMED"},
			expected:    map[string]string{"FOO": "bar"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			valueFile := ""
			if test.fileContent != "" {
				dir := t.NewTempDir().Root()
				valueFile = filepath.Join(dir, "values.env")
				if err := os.WriteFile(valueFile, []byte(test.fileContent), 0o600); err != nil {
					t.Fatalf("writing temp value file: %v", err)
				}
			}

			got, err := mergeDeployParams(test.base, valueFile, test.overrides)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, got)
		})
	}
}

func TestMergeDeployParams_ValueFileNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		_, err := mergeDeployParams(nil, "/does/not/exist.env", nil)
		t.CheckError(true, err)
	})
}

// TestGetActionsRunner_InjectsDeployParamsIntoExecEnv exercises the end-to-end
// wiring: --set and --set-value-file values from SkaffoldOptions must reach
// the docker exec-env constructor as part of its envMap argument, so that
// downstream container tasks receive them as environment variables.
func TestGetActionsRunner_InjectsDeployParamsIntoExecEnv(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.NewTempDir().Root()
		valueFile := filepath.Join(dir, "values.env")
		if err := os.WriteFile(valueFile, []byte("FROM_FILE=file\nCOMMON=file\n"), 0o600); err != nil {
			t.Fatalf("writing temp value file: %v", err)
		}

		runCtx := runcontext.RunContext{
			Opts: config.SkaffoldOptions{
				ManifestsOverrides: []string{"FROM_SET=cli", "COMMON=cli"},
				ManifestsValueFile: valueFile,
			},
			Pipelines: runcontext.NewPipelines(map[string]latest.Pipeline{
				"default": {
					CustomActions: []latest.Action{{
						Name: "act",
						ExecutionModeConfig: latest.ActionExecutionModeConfig{
							VerifyExecutionModeType: latest.VerifyExecutionModeType{
								LocalExecutionMode: &latest.LocalVerifier{},
							},
						},
					}},
				},
			}, []string{"default"}),
		}

		var capturedEnv map[string]string
		t.Override(&docker.NewExecEnv, func(_ context.Context, _ dockerutil.Config, _ *label.DefaultLabeller, _ []*latest.PortForwardResource, _ string, envMap map[string]string, _ []latest.Action) (*docker.ExecEnv, error) {
			capturedEnv = envMap
			return &docker.ExecEnv{}, nil
		})

		_, err := GetActionsRunner(context.TODO(), &runCtx, &label.DefaultLabeller{}, "", "")
		t.CheckNoError(err)

		t.CheckDeepEqual("file", capturedEnv["FROM_FILE"])
		t.CheckDeepEqual("cli", capturedEnv["FROM_SET"])
		// --set overrides --set-value-file on key collision.
		t.CheckDeepEqual("cli", capturedEnv["COMMON"])
	})
}
