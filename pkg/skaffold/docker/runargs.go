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
	"fmt"
	"strings"

	"github.com/moby/moby/api/types/container"
)

// allowedRunArgs is the conservative whitelist of `docker run` flags that may
// be forwarded to a local container via `runArgs`. Every accepted flag maps to
// a field on the Docker HostConfig.
const allowedRunArgs = "--network, -v/--volume, --add-host, --tmpfs"

// RunArgs is the parsed, whitelisted projection of a user-supplied
// executionMode.local.runArgs list (shared by custom actions and verify).
//
// Only a small, deliberately conservative subset of `docker run` flags is
// recognised; unknown flags are rejected so users fail fast instead of being
// silently ignored. See ParseRunArgs for the full list.
type RunArgs struct {
	NetworkMode string
	Binds       []string
	ExtraHosts  []string
	Tmpfs       map[string]string
}

// ParseRunArgs parses a docker-run-style argument list and returns a
// whitelisted RunArgs projection. Supported flags:
//
//	--network=VALUE
//	-v=SRC:DST[:MODE]   (also --volume=...)
//	--add-host=HOST:IP
//	--tmpfs=PATH[:OPTIONS]
//
// Each flag must be in the `--flag=value` (or `-f=value`) form; the
// space-separated variant is not supported to keep the parser unambiguous.
// A nil / empty input returns a nil *RunArgs.
func ParseRunArgs(args []string) (*RunArgs, error) {
	if len(args) == 0 {
		return nil, nil
	}
	out := &RunArgs{}
	for i, raw := range args {
		arg := strings.TrimSpace(raw)
		if arg == "" {
			continue
		}
		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			// No '=' means either an unknown bare flag or the space-separated form.
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("runArgs[%d] %q: unsupported flag %q (only --flag=value form is supported; allowed: %s)", i, raw, arg, allowedRunArgs)
			}
			return nil, fmt.Errorf("runArgs[%d] %q: only --flag=value form is supported (no space-separated values)", i, raw)
		}
		switch key {
		case "--network":
			out.NetworkMode = val
		case "-v", "--volume":
			out.Binds = append(out.Binds, val)
		case "--add-host":
			out.ExtraHosts = append(out.ExtraHosts, val)
		case "--tmpfs":
			if out.Tmpfs == nil {
				out.Tmpfs = map[string]string{}
			}
			mountPath, opts, _ := strings.Cut(val, ":")
			out.Tmpfs[mountPath] = opts
		default:
			return nil, fmt.Errorf("runArgs[%d] %q: unsupported flag %q (allowed: %s)", i, raw, key, allowedRunArgs)
		}
	}
	return out, nil
}

// ApplyToHostConfig overlays parsed runArgs onto hc in place. NetworkMode is
// only overridden when the user provided one. A nil receiver is a no-op.
func (r *RunArgs) ApplyToHostConfig(hc *container.HostConfig) {
	if r == nil || hc == nil {
		return
	}
	if r.NetworkMode != "" {
		hc.NetworkMode = container.NetworkMode(r.NetworkMode)
	}
	if len(r.Binds) > 0 {
		hc.Binds = append(hc.Binds, r.Binds...)
	}
	if len(r.ExtraHosts) > 0 {
		hc.ExtraHosts = append(hc.ExtraHosts, r.ExtraHosts...)
	}
	if len(r.Tmpfs) > 0 {
		if hc.Tmpfs == nil {
			hc.Tmpfs = map[string]string{}
		}
		for k, v := range r.Tmpfs {
			hc.Tmpfs[k] = v
		}
	}
}
