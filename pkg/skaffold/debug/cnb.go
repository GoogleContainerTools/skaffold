/*
Copyright 2020 The Skaffold Authors

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

package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cnbl "github.com/buildpacks/lifecycle/launch"
	cnb "github.com/buildpacks/lifecycle/platform"
	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

const (
	// cnbLauncher is the location of the CNB lifecycle launcher.
	cnbLauncher = "/cnb/lifecycle/launcher"

	// New in Platform API 0.4, the CNB lifecycle launcher creates executables
	// in `/cnb/process` that launch the corresponding process name.
	cnbProcessLauncherPrefix = "/cnb/process/"
)

func init() {
	// We rewrite CNB entrypoints to use the cnbLauncher, and unwrap the process
	// definitions to launch the configured command-line.  So our debug transforms
	// can ignore the launcher.
	entrypointLaunchers = append(entrypointLaunchers, cnbLauncher)
}

// isCNBImage returns true if this image is a CNB-produced image.
// CNB images use a special launcher as the entrypoint. In CNB Platform API 0.3,
// this was always `/cnb/lifecycle/launcher`, but Platform API 0.4 (introduced in pack 0.13)
// allows using a symlink to a file in `/cnb/process/<type>`.  More below.
func isCNBImage(ic ImageConfiguration) bool {
	if _, found := ic.Labels["io.buildpacks.stack.id"]; !found {
		return false
	}
	return len(ic.Entrypoint) == 1 && (ic.Entrypoint[0] == cnbLauncher || strings.HasPrefix(ic.Entrypoint[0], cnbProcessLauncherPrefix))
}

// hasCNBLauncherEntrypoint returns true if the entrypoint is the cnbLauncher.
func hasCNBLauncherEntrypoint(ic ImageConfiguration) bool {
	return len(ic.Entrypoint) == 1 && ic.Entrypoint[0] == cnbLauncher
}

// updateForCNBImage normalizes a CNB image by rewriting the CNB launch configuration into
// a more traditional entrypoint and arguments, prior to handing off to another
// transformer.  This transformer is usually the `debug` container transform process.
// After the transformer, updateForCNBImage rewrites the altered command-line back to
// the form expected by the CNB launcher.
//
// CNB images use a special launcher executable as the entrypoint.  This launcher sets up the
// execution environment as configured by the various buildpacks, and then hands off to the
// configured launch.  The CNB launcher supports three types of launches:
//
//  1. _predefined processes_ are named sets of command+arguments (similar to a container image's
//     ENTRYPOINT/CMD).  Processes are created by the buildpacks, and oftentimes there is a
//     buildpack that parses a user's `Procfile` and turns those contents into processes.
//     There are two types of process definitions:
//     - *direct*: these are passed uninterpreted to os.exec; the command is resolved in PATH
//     Note that in practice we see direct-style definitions that execute `/bin/sh -c ...`
//     - *script*: the command is treated as a shell script and passed to `sh -c`, and any remaining
//     arguments on the container command-line are added to the shell and so become available
//     as positional arguments (see https://github.com/buildpacks/lifecycle/issues/218#issuecomment-567091462).
//     For example: `sh -c 'echo $0 $1 $2 $3' arg0 arg1 arg2 arg3` => `arg0 arg1 arg2 arg3`.
//  2. _direct execs_: the user can provide a command-line which is treated like a _direct process_.
//  3. _shell scripts_: the user can provide a shell script as the first argument and any
//     remaining arguments are available as positional arguments like _script processes_.
//
// Script-style launches support referencing environment variables since they are expanded by the shell.
//
// Configuring the launch depends on the CNB Platform API version being used, which is determined by
// the builder's lifecycle version, which is itself determined by the pack used to create a builder.
//   - In Platform API 0.3 (pack 0.12 and earlier / lifecycle 0.8 and earlier) the image entrypoint
//     is set to `/cnb/lifecycle/launcher`.  The launch is determined by:
//     1. If there are arguments:
//     1. If there is a single argument and it matches a process type, then the corresponding
//     process is launched.
//     2. If the first argument is `--` then the remaining arguments are treated as a _direct exec_.
//     3. Otherwise the first argument is treated as a shell script launch with the first
//     argument as the script and remaining arguments are positional arguments to the script.
//     2. If there are no arguments, a process type is taken from the `CNB_PROCESS_TYPE`
//     environment variable, defaulting to `web`.
//   - In Platform API 0.4 (pack 0.13 / lifecycle 0.9) the process types are turned into executables
//     found in `/cnb/process/`, and the image entrypoint is set to the corresponding executable for
//     the default process type.  `CNB_PROCESS_TYPE` is ignored in this situation.  A different process
//     can be used by overriding the image entrypoint.  Direct and script launches are supported by
//     setting the entrypoint to `/cnb/lifecycle/launcher` and providing the appropriate arguments.
func updateForCNBImage(adapter types.ContainerAdapter, ic ImageConfiguration, transformer func(adapter types.ContainerAdapter, ic ImageConfiguration) (types.ContainerDebugConfiguration, string, error)) (types.ContainerDebugConfiguration, string, error) {
	// buildpacks/lifecycle 0.6.0 embeds the process definitions into a special image label.
	// The build metadata isn't absolutely required as the image args could be
	// a command line (e.g., `python xxx`) but it likely indicates the
	// image was built with an older lifecycle.
	metadataJSON, found := ic.Labels["io.buildpacks.build.metadata"]
	if !found {
		return types.ContainerDebugConfiguration{}, "", fmt.Errorf("image is missing buildpacks metadata; perhaps built with older lifecycle?")
	}
	m := cnb.BuildMetadata{}
	if err := json.Unmarshal([]byte(metadataJSON), &m); err != nil {
		return types.ContainerDebugConfiguration{}, "", fmt.Errorf("unable to parse image buildpacks metadata")
	}
	if len(m.Processes) == 0 {
		return types.ContainerDebugConfiguration{}, "", fmt.Errorf("buildpacks metadata has no processes")
	}

	needsCnbLauncher := ic.Entrypoint[0] != cnbLauncher
	// Rewrites the command-line with cnbLauncher as the entrypoint
	ic, rewriter := adjustCommandLine(m, ic)

	// The CNB launcher uses CNB_APP_DIR (defaults to /workspace) and ignores the image's working directory.
	if appDir := ic.Env["CNB_APP_DIR"]; appDir != "" {
		ic.WorkingDir = appDir
	} else {
		ic.WorkingDir = "/workspace"
	}

	c, img, err := transformer(adapter, ic)
	if err != nil {
		return c, img, err
	}
	// must explicitly modify the working dir as the imageConfig is lost after we return
	if c.WorkingDir == "" {
		c.WorkingDir = ic.WorkingDir
	}

	container := adapter.GetContainer()
	if container.Args != nil && rewriter != nil {
		// Only rewrite the container if the arguments were changed: some transforms only alter
		// env vars, and the image's arguments are not changed.
		if needsCnbLauncher {
			container.Command = []string{cnbLauncher}
		}
		container.Args = rewriter(container.Args)
	}
	return c, img, err
}

// adjustCommandLine resolves the launch process and then rewrites the command-line to be
// in a form suitable for the normal `skaffold debug` transformations.  It returns an
// amended configuration with a function to re-transform the command-line to the form
// expected by cnbLauncher.
func adjustCommandLine(m cnb.BuildMetadata, ic ImageConfiguration) (ImageConfiguration, func([]string) []string) {
	// check for direct exec
	if hasCNBLauncherEntrypoint(ic) && len(ic.Arguments) > 0 && ic.Arguments[0] == "--" {
		// strip and then restore the "--"
		ic.Arguments = ic.Arguments[1:]
		return ic, func(transformed []string) []string {
			return append([]string{"--"}, transformed...)
		}
	}

	if p, clArgs, found := findCNBProcess(ic, m); found {
		// Direct: p.Command is the command and p.Args are the arguments
		if p.Direct {
			// Detect and unwrap `/bin/sh -c ...`-style command lines.
			// For example, GCP Buildpacks turn Procfiles into `/bin/bash -c ...`
			if len(p.Args) >= 2 && isShDashC(p.Command, p.Args[0]) {
				p.Command = p.Args[1]
				p.Args = p.Args[2:]
				// and fall through to script type below
			} else {
				args := append([]string{p.Command}, p.Args...)
				args = append(args, clArgs...)
				ic.Entrypoint = []string{cnbLauncher}
				ic.Arguments = args
				return ic, func(transformed []string) []string {
					return append([]string{"--"}, transformed...)
				}
			}
		}
		// Script type: split p.Command, pass it through the transformer, and then reassemble in the rewriter.
		ic.Entrypoint = []string{cnbLauncher}
		if args, err := shell.Split(p.Command); err == nil {
			ic.Arguments = args
		} else {
			ic.Arguments = []string{p.Command}
		}
		return ic, func(transformed []string) []string {
			// reassemble back into a script with arguments
			result := append([]string{shJoin(transformed)}, p.Args...)
			result = append(result, clArgs...)
			return result
		}
	}

	if len(ic.Arguments) == 0 {
		log.Entry(context.TODO()).Warnf("no CNB launch found for %s", ic.Artifact)
		return ic, nil
	}

	// ic.arguments[0] is a shell script:  split it, pass it through the transformer, and then reassemble in the rewriter.
	// If it can't be split, then we return it untouched, to be handled by the normal debug process.
	if cmdline, err := shell.Split(ic.Arguments[0]); err == nil {
		positionals := ic.Arguments[1:] // save aside the script positional arguments
		ic.Arguments = cmdline
		return ic, func(transformed []string) []string {
			// reassemble back into a script with the positional arguments
			return append([]string{shJoin(transformed)}, positionals...)
		}
	}
	return ic, nil
}

// findCNBProcess tries to resolve a CNB process definition given the image configuration.
// It is assumed that the image is a CNB image.
func findCNBProcess(ic ImageConfiguration, m cnb.BuildMetadata) (cnbl.Process, []string, bool) {
	if hasCNBLauncherEntrypoint(ic) && len(ic.Arguments) > 0 {
		// the launcher accepts the first argument as a process type
		if len(ic.Arguments) == 1 {
			processType := ic.Arguments[0]
			for _, p := range m.Processes {
				if p.Type == processType {
					return p, nil, true // drop the argument
				}
			}
		}
		return cnbl.Process{}, nil, false
	}

	// determine process-type
	processType := "web" // default buildpacks process type
	platformAPI := ic.Env["CNB_PLATFORM_API"]
	if platformAPI == "0.4" {
		// Platform API 0.4-style /cnb/process/xxx
		if !strings.HasPrefix(ic.Entrypoint[0], cnbProcessLauncherPrefix) {
			return cnbl.Process{}, nil, false
		}
		processType = ic.Entrypoint[0][len(cnbProcessLauncherPrefix):]
	} else if len(ic.Env["CNB_PROCESS_TYPE"]) > 0 {
		processType = ic.Env["CNB_PROCESS_TYPE"]
	}

	for _, p := range m.Processes {
		if p.Type == processType {
			return p, ic.Arguments, true
		}
	}
	return cnbl.Process{}, nil, false
}
