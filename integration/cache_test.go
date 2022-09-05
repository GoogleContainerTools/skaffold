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

package integration

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCacheAPITriggers(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("examples/getting-started").RunOrFail(t)

	ns, _ := SetupNamespace(t)
	rpcAddr := randomPort()

	// Disable caching to ensure we get a "build in progress" event each time.
	skaffold.Dev("--cache-artifacts=false", "--rpc-port", rpcAddr).InDir("examples/getting-started").InNs(ns.Name).RunBackground(t)

	// Ensure we see a build triggered in the event log
	_, entries := apiEvents(t, rpcAddr)

	waitForEvent(t, entries, func(e *proto.LogEntry) bool {
		return e.GetEvent().GetBuildEvent().GetArtifact() == "skaffold-example"
	})
}

func TestCacheHits(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	testutil.Run(t, "TestCacheHits", func(t *testutil.T) {
		// Run skaffold build first to fail quickly on a build failure
		skaffold.Build().InDir("examples/getting-started").RunOrFail(t.T)

		ns, _ := SetupNamespace(t.T)
		rpcAddr := randomPort()

		// Rebuild with a different tag so that we get a cache hit.
		out := skaffold.Build("--tag", ns.Name, "--rpc-port", rpcAddr).InDir("examples/getting-started").RunOrFailOutput(t.T)
		t.CheckContains("skaffold-example: Found. Tagging", string(out))
	})
}

func waitForEvent(t *testing.T, entries chan *proto.LogEntry, condition func(*proto.LogEntry) bool) {
	failNowIfError(t, wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) { return condition(<-entries), nil }))
}

func TestCacheIfBuildFail(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	cacheFile := "cache_" + ns.Name
	testDir := "testdata/cache"
	relativePath := path.Join(testDir, cacheFile)
	defer os.Remove(relativePath)

	skaffold.Build("--cache-file", cacheFile).InDir(testDir).InNs(ns.Name).Run(t)

	fInfo, err := os.Stat(relativePath)
	failNowIfError(t, err)
	if b := fInfo.Size(); b == 0 {
		failNowIfError(t, fmt.Errorf("expected to see content in the cache file, saw %d bytes", b))
	}
}
