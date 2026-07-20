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

package cmd

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestNewCmdDelete(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Chdir()
		t.Override(&opts, config.SkaffoldOptions{})
		t.Override(&dryRun, false)

		cmd := NewCmdDelete()
		cmd.SilenceUsage = true

		// Check that the command accepts the --set flag and it's correctly bound
		err := cmd.Flags().Parse([]string{"--set", "key1=val1", "--set", "key2=val2"})
		t.CheckNoError(err)
		t.CheckDeepEqual([]string{"key1=val1", "key2=val2"}, opts.ManifestsOverrides)

		// Check that the command accepts the --set-value-file flag and it's correctly bound
		err = cmd.Flags().Parse([]string{"--set-value-file", "values.env"})
		t.CheckNoError(err)
		t.CheckDeepEqual("values.env", opts.ManifestsValueFile)

		// Check that the command accepts the --dry-run flag and it's correctly bound
		err = cmd.Flags().Parse([]string{"--dry-run"})
		t.CheckNoError(err)
		t.CheckTrue(dryRun)
	})
}
