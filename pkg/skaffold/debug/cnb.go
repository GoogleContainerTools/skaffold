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

// updateForCNBImage transforms an imageConfiguration for a Cloud Native Buildpacks-created image
// for the configured process prior to handing off to another transformer.
func updateForCNBImage(container *v1.Container, ic imageConfiguration, transformer func(container *v1.Container, ic imageConfiguration) (ContainerDebugConfiguration, string, error)) (ContainerDebugConfiguration, string, error) {
	processType := "web"
	if value, found := ic.env["CNB_PROCESS_TYPE"]; found && len(value) > 0 {
		processType = value
	}
	m := cnb.BuildMetadata{}
	// buildpacks/lifecycle 0.6.0 now embeds processes into special image label
	if metadataJSON, found := ic.labels["io.buildpacks.build.metadata"]; !found {
		return ContainerDebugConfiguration{}, "", fmt.Errorf("buildpacks build metadata not present; image built with older lifecycle?")
	} else {
		json.Unmarshal([]byte(metadataJSON), &m)
	}
	if len(m.Processes) == 0 {
		return ContainerDebugConfiguration{}, "", fmt.Errorf("buildpacks build metadata is missing processes metadata")
	}

	for _, p := range m.Processes {
		// the launcher accepts the first argument as a process type
		if p.Type == processType || (len(ic.arguments) == 1 && p.Type == ic.arguments[0]) {
			logrus.Debugf("Setting command for %q to %q process: %q + %q\n", ic.artifact, processType, p.Command, p.Args)
			// retain the buildpacks launcher as the entrypoint
			if p.Direct {
				ic.arguments = append([]string{p.Command}, p.Args...)
			} else {
				// p.Command is a shell script executed via `sh -c "..."`, and p.Args are added as arguments to the script
				// https://github.com/buildpacks/lifecycle/issues/218#issuecomment-567091462
				if args, err := shell.Split(p.Command); err == nil {
					ic.arguments = args
				} else {
					ic.arguments = []string{p.Command}
				}
			}
			c, img, err := transformer(container, ic)
			if err == nil {
				if p.Direct {
					container.Args = append([]string{"--"}, container.Args...)
				} else {
					// Args[0] is a shell script for sh -c
					// Args[1:]] are arguments to that shell script
					container.Args = append([]string{shJoin(container.Args)}, p.Args...)
				}
			}
			return c, img, err
		}
	}
	return ContainerDebugConfiguration{}, "", fmt.Errorf("could not find buildpack process of type %q", processType)
}
