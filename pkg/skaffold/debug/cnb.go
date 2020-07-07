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
	"encoding/json"
	"fmt"

	cnb "github.com/buildpacks/lifecycle"
	shell "github.com/kballard/go-shellquote"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func init() {
	// the CNB's launcher just launches the command
	entrypointLaunchers = append(entrypointLaunchers, "/cnb/lifecycle/launcher")
}

// updateForCNBImage normalizes an imageConfiguration from a Cloud Native Buildpacks-created image
// prior to handing off to another transformer.  This transformer is usually the `debug` container
// transform process.  CNB images have their entrypoint set to the CNB launcher, and the image command
// describe the launch parameters.  After the transformer, updateForCNBImage rewrites the changed
// image command back to the form expected by the CNB launcher.
//
// The CNB launcher supports three types of launches:
//
//   1. _predefined processes_ are named sets of command+arguments.  There are two types:
//      - direct: these are passed uninterpreted to os.exec; the command is resolved in PATH
//        Note that these may actually be configured to execute `/bin/sh -c ...`
//      - script: the command is treated as a shell script and passed to `sh -c`, the remaining
//        arguments are added to the shell, and so available as positional arguments.
//        For example: `sh -c 'echo $0 $1 $2 $3' arg0 arg1 arg2 arg3` => `arg0 arg1 arg2 arg3`.
//        (https://github.com/buildpacks/lifecycle/issues/218#issuecomment-567091462).
//   2. _direct execs_ where the container's arg[0] == `--` are treated like direct processes
//   3. _shell scripts_ where the container's arg[0] is the script, and are treated are like indirect processes.
//
// A key point is that the script-style launches allow support references to environment variables.
// So we find the command line to be executed, whether that is a script or a direct, and turn it into
// a normal command-line.  But we also return a rewriter to transform the the command-line back to
// the original form.
func updateForCNBImage(container *v1.Container, ic imageConfiguration, transformer func(container *v1.Container, ic imageConfiguration) (ContainerDebugConfiguration, string, error)) (ContainerDebugConfiguration, string, error) {
	// The build metadata isn't absolutely required as the image args could be
	// a command line (e.g., `python xxx`) but it likely indicates the
	// image was built with an older lifecycle.
	// buildpacks/lifecycle 0.6.0 now embeds processes into special image label
	metadataJSON, found := ic.labels["io.buildpacks.build.metadata"]
	if !found {
		return ContainerDebugConfiguration{}, "", fmt.Errorf("image is missing buildpacks metadata; perhaps built with older lifecycle?")
	}
	m := cnb.BuildMetadata{}
	if err := json.Unmarshal([]byte(metadataJSON), &m); err != nil {
		return ContainerDebugConfiguration{}, "", fmt.Errorf("unable to parse image buildpacks metadata")
	}
	if len(m.Processes) == 0 {
		return ContainerDebugConfiguration{}, "", fmt.Errorf("buildpacks metadata has no processes")
	}

	// The CNB launcher is retained as the entrypoint.
	ic, rewriter := adjustCommandLine(m, ic)

	// The CNB launcher uses CNB_APP_DIR (defaults to /workspace) and ignores the image's working directory.
	if appDir := ic.env["CNB_APP_DIR"]; appDir != "" {
		ic.workingDir = appDir
	} else {
		ic.workingDir = "/workspace"
	}

	c, img, err := transformer(container, ic)
	// must explicitly modify the working dir as the imageConfig is lost after we return
	if c.WorkingDir == "" {
		c.WorkingDir = ic.workingDir
	}

	// Only rewrite the container.Args if set: some transforms only alter env vars,
	// and the image's arguments are not changed.
	if err == nil && container.Args != nil && rewriter != nil {
		container.Args = rewriter(container.Args)
	}
	return c, img, err
}

// adjustCommandLine resolves the launch process and then rewrites the command-line to be
// in a form suitable for the normal `skaffold debug` transformations.  It returns an
// amended configuration with a function to re-transform the command-line to the form
// expected by the launcher.
func adjustCommandLine(m cnb.BuildMetadata, ic imageConfiguration) (imageConfiguration, func([]string) []string) {
	// direct exec
	if len(ic.arguments) > 0 && ic.arguments[0] == "--" {
		// strip and restore the "--"
		ic.arguments = ic.arguments[1:]
		return ic, func(transformed []string) []string {
			return append([]string{"--"}, transformed...)
		}
	}

	processType := "web" // default buildpacks process type
	// the launcher accepts the first argument as a process type
	if len(ic.arguments) == 1 {
		processType = ic.arguments[0]
	} else if value := ic.env["CNB_PROCESS_TYPE"]; len(value) > 0 {
		processType = value
	}

	for _, p := range m.Processes {
		if p.Type == processType {
			// Direct: p.Command is the command and p.Args are the arguments
			if p.Direct {
				// Detect and unwrap `/bin/sh -c ...`-style command lines; GCP Buildpacks turn Procfiles into `/bin/bash -c ...`
				if len(p.Args) >= 2 && isShDashC(p.Command, p.Args[0]) {
					p.Command = p.Args[1]
					p.Args = p.Args[2:]
					// and fall through to script type below
				} else {
					ic.arguments = append([]string{p.Command}, p.Args...)
					return ic, func(transformed []string) []string {
						return append([]string{"--"}, transformed...)
					}
				}
			}
			// Script type: split p.Command, pass it through the transformer, and then reassemble in the rewriter.
			if args, err := shell.Split(p.Command); err == nil {
				ic.arguments = args
			} else {
				ic.arguments = []string{p.Command}
			}
			return ic, func(transformed []string) []string {
				// reassemble back into a script with arguments
				return append([]string{shJoin(transformed)}, p.Args...)
			}
		}
	}

	if len(ic.arguments) == 0 {
		// indicates an image mis-configuration as we should have resolved the the
		// CNB_PROCESS_TYPE (if specified) or `web`.
		logrus.Warnf("no CNB launch found for %s/%s", ic.artifact, processType)
		return ic, nil
	}

	// ic.arguments[0] is a shell script:  split it, pass it through the transformer, and then reassemble in the rewriter.
	// If it can't be split, then we fall through and return it untouched, to be handled by the normal debug process.
	var rewriter func(transformed []string) []string
	if args, err := shell.Split(ic.arguments[0]); err == nil {
		remnants := ic.arguments[1:]
		ic.arguments = args
		rewriter = func(transformed []string) []string {
			// reassemble back into a script with arguments
			return append([]string{shJoin(transformed)}, remnants...)
		}
	}
	return ic, rewriter
}
