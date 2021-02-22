/*
Copyright 2021 The Skaffold Authors

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

package custom

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewRunner(t *testing.T) {
	const (
		imageName = "foo.io/baz"
	)

	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image "+imageName+" --config "+tmpDir.Path("test.yaml")))

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
				CustomTests: []latest.CustomTest{{
					Command:        "./build.sh",
					TimeoutSeconds: "10",
					Dependencies: &latest.CustomTestDependencies{
						Command: "echo [\"file1\",\"file2\",\"file3\"]",
						Paths:   []string{"**"},
						Ignore:  []string{"b*"},
					},
				}},
			}},
		}

		custom := latest.CustomTest{
			Command:        "./test.sh",
			TimeoutSeconds: "10",
			Dependencies: &latest.CustomTestDependencies{
				Command: "echo [\"file1\",\"file2\",\"file3\"]",
				Paths:   []string{"**"},
				Ignore:  []string{"b*"},
			},
		}

		testRunner, err := New(cfg, cfg.workingDir, custom)
		t.CheckNoError(err)
		err = testRunner.Test(context.Background(), ioutil.Discard, nil)
		t.CheckNoError(err)
	})
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	workingDir            string
	tests                 []*latest.TestCase
	muted                 config.Muted
}
