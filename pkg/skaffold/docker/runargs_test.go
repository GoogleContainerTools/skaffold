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

package docker

import (
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestParseRunArgs_Empty(t *testing.T) {
	got, err := ParseRunArgs(nil)
	testutil.CheckError(t, false, err)
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}

	got, err = ParseRunArgs([]string{})
	testutil.CheckError(t, false, err)
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestParseRunArgs_Supported(t *testing.T) {
	got, err := ParseRunArgs([]string{
		"--network=host",
		"-v=/host:/container:ro",
		"--volume=/data:/data",
		"--add-host=db:127.0.0.1",
		"--tmpfs=/tmp:size=64m",
	})
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "host", got.NetworkMode)
	testutil.CheckDeepEqual(t, []string{"/host:/container:ro", "/data:/data"}, got.Binds)
	testutil.CheckDeepEqual(t, []string{"db:127.0.0.1"}, got.ExtraHosts)
	testutil.CheckDeepEqual(t, map[string]string{"/tmp": "size=64m"}, got.Tmpfs)
}

func TestParseRunArgs_UnsupportedFlag(t *testing.T) {
	// Flags that used to be on the whitelist must now be rejected.
	for _, flag := range []string{"--rm", "-e=FOO=bar", "--user=1000", "--privileged", "--cap-add=NET_ADMIN"} {
		_, err := ParseRunArgs([]string{flag})
		if err == nil || !strings.Contains(err.Error(), "unsupported flag") {
			t.Fatalf("%q: expected unsupported flag error, got %v", flag, err)
		}
	}
}

func TestParseRunArgs_SpaceSeparated(t *testing.T) {
	_, err := ParseRunArgs([]string{"plain value"})
	if err == nil || !strings.Contains(err.Error(), "only --flag=value form") {
		t.Fatalf("expected only --flag=value error, got %v", err)
	}
}

func TestParseRunArgs_SkipsEmptyEntries(t *testing.T) {
	got, err := ParseRunArgs([]string{"", "   ", "--network=host"})
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "host", got.NetworkMode)
}

func TestApplyToHostConfig_NilSafe(t *testing.T) {
	var r *RunArgs
	r.ApplyToHostConfig(nil)
	r.ApplyToHostConfig(&container.HostConfig{})
	(&RunArgs{}).ApplyToHostConfig(nil)
}

func TestApplyToHostConfig_OverridesAndAppends(t *testing.T) {
	r := &RunArgs{
		NetworkMode: "host",
		Binds:       []string{"/a:/a"},
		ExtraHosts:  []string{"h:1.2.3.4"},
		Tmpfs:       map[string]string{"/tmp": "size=16m"},
	}
	hc := &container.HostConfig{
		NetworkMode: container.NetworkMode("bridge"),
		Binds:       []string{"/pre:/pre"},
		ExtraHosts:  []string{"pre:0.0.0.0"},
		Tmpfs:       map[string]string{"/run": "size=8m"},
	}
	r.ApplyToHostConfig(hc)
	testutil.CheckDeepEqual(t, "host", string(hc.NetworkMode))
	testutil.CheckDeepEqual(t, []string{"/pre:/pre", "/a:/a"}, hc.Binds)
	testutil.CheckDeepEqual(t, []string{"pre:0.0.0.0", "h:1.2.3.4"}, hc.ExtraHosts)
	testutil.CheckDeepEqual(t, map[string]string{"/run": "size=8m", "/tmp": "size=16m"}, hc.Tmpfs)
}

func TestApplyToHostConfig_EmptyRunArgsDoesNotOverrideNetwork(t *testing.T) {
	r := &RunArgs{}
	hc := &container.HostConfig{NetworkMode: container.NetworkMode("bridge")}
	r.ApplyToHostConfig(hc)
	testutil.CheckDeepEqual(t, "bridge", string(hc.NetworkMode))
}
