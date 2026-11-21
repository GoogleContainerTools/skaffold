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

package helm

import (
	"context"
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const testPluginDir = "TEMPORARY-TEST-DIR/PLUGIN-NAME"

// The plugin manifest depends only on the Skaffold binary path — everything that
// varies per release travels in SKAFFOLD_CMDLINE — so it is installed once and
// reused for the whole run. Installing per call cost ~0.54s per cycle, paid once
// per release on the deploy path and once per config on the render path: 21
// installs + 21 uninstalls on a 21-release monorepo.
//
// CmdRun asserts the EXACT command sequence, so a single expected
// `helm plugin install` across five calls is the assertion.
func TestPreparePostRendererInstallsOncePerProcess(t *testing.T) {
	testutil.Run(t, "installs once across many calls", func(t *testutil.T) {
		ResetSharedPostRendererForTest()
		t.Cleanup(ResetSharedPostRendererForTest)
		t.Override(&PluginInstallDir, testPluginDir)
		t.Override(&util.DefaultExecCommand,
			testutil.CmdRun("helm plugin install "+testPluginDir))

		helm4 := semver.MustParse("4.2.3")
		var first []string
		for i := 0; i < 5; i++ {
			cleanup, args, err := PreparePostRenderer(
				context.Background(), mockClient{}, "/path/to/skaffold", helm4)
			t.CheckNoError(err)
			// A non-nil cleanup would uninstall the shared plugin the moment the
			// caller's defer ran, reinstating the per-call install.
			t.CheckDeepEqual(true, cleanup == nil)
			if i == 0 {
				first = args
				continue
			}
			t.CheckDeepEqual(first, args)
		}
		t.CheckDeepEqual("--post-renderer", first[0])
	})
}

// A different Skaffold binary means a different plugin manifest, so the cache
// must not hand back the old plugin.
func TestPreparePostRendererReinstallsForDifferentBinary(t *testing.T) {
	testutil.Run(t, "reinstalls when the binary changes", func(t *testutil.T) {
		ResetSharedPostRendererForTest()
		t.Cleanup(ResetSharedPostRendererForTest)
		t.Override(&PluginInstallDir, testPluginDir)
		t.Override(&util.DefaultExecCommand, testutil.
			CmdRun("helm plugin install "+testPluginDir).
			AndRun("helm plugin install "+testPluginDir))

		helm4 := semver.MustParse("4.2.3")
		_, _, err := PreparePostRenderer(context.Background(), mockClient{}, "/bin/one", helm4)
		t.CheckNoError(err)
		_, _, err = PreparePostRenderer(context.Background(), mockClient{}, "/bin/two", helm4)
		t.CheckNoError(err)
		t.CheckDeepEqual("/bin/two", sharedPostRenderer.binary)
	})
}

// Helm v3 takes the executable directly and must never install a plugin.
func TestPreparePostRendererHelm3NeedsNoPlugin(t *testing.T) {
	ResetSharedPostRendererForTest()
	t.Cleanup(ResetSharedPostRendererForTest)

	cleanup, args, err := PreparePostRenderer(
		context.Background(), mockClient{}, "/path/to/skaffold", semver.MustParse("3.19.0"))
	if err != nil {
		t.Fatal(err)
	}
	if cleanup != nil {
		t.Error("helm v3 must not return a plugin cleanup")
	}
	if len(args) != 2 || args[0] != "--post-renderer" || args[1] != "/path/to/skaffold" {
		t.Errorf("helm v3 should pass the binary directly, got %v", args)
	}
	if sharedPostRenderer.name != "" {
		t.Error("helm v3 must not install a shared plugin")
	}
}

func TestCleanupSharedPostRendererIsSafeWhenNothingInstalled(t *testing.T) {
	ResetSharedPostRendererForTest()
	// Must not panic and must not shell out to helm.
	CleanupSharedPostRenderer(context.Background())
}
