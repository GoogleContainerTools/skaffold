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

package debug

import (
	"context"
	"fmt"

	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

// containerTransforms are the set of configured transformers
var containerTransforms []containerTransformer

// RegisterContainerTransformer allows calling packages to register their own
// transformer implementation with the global transform list.
// It returns a function to reset the transform list to its original state,
// which is useful only for testing.
func RegisterContainerTransformer(t containerTransformer) func() {
	cpy := make([]containerTransformer, len(containerTransforms))
	copy(cpy, containerTransforms)
	resetFunc := func() { containerTransforms = cpy }
	containerTransforms = append(containerTransforms, t)
	return resetFunc
}

// containerTransformer transforms a container definition
type containerTransformer interface {
	// IsApplicable determines if this container is suitable to be transformed.
	IsApplicable(config ImageConfiguration) bool

	// Apply configures a container definition for debugging, returning the debug configuration details
	// and required initContainer (an empty string if not required), or return a non-nil error if
	// the container could not be transformed.  The initContainer image is intended to install any
	// required debug support tools.
	Apply(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator, overrideProtocols []string) (types.ContainerDebugConfiguration, string, error)
}

// TransformContainer rewrites the container definition to enable debugging.
// Returns a debugging configuration description with associated language runtime support
// container image, or an error if the rewrite was unsuccessful.
func TransformContainer(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator) (types.ContainerDebugConfiguration, string, error) {
	configuration, requiredImage, err := transformContainer(adapter, config, portAlloc)
	if err == nil {
		configuration.Artifact = config.Artifact
		if configuration.WorkingDir == "" {
			configuration.WorkingDir = config.WorkingDir
		}
	}
	return configuration, requiredImage, err
}

func transformContainer(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator) (types.ContainerDebugConfiguration, string, error) {
	// Update the image configuration's environment with those set in the k8s manifest.
	// (Environment variables in the k8s container's `env` add to the image configuration's `env` settings rather than replace.)
	container := adapter.GetContainer()
	defer adapter.Apply()
	for _, key := range container.Env.Order {
		if config.Env == nil {
			config.Env = make(map[string]string)
		}
		config.Env[key] = container.Env.Env[key]
	}

	if len(container.Command) > 0 {
		config.Entrypoint = container.Command
	}
	if len(container.Args) > 0 {
		config.Arguments = container.Args
	}

	// Apply command-line unwrapping for buildpack images and images using `sh -c`-style command-lines
	next := func(adapter types.ContainerAdapter, config ImageConfiguration) (types.ContainerDebugConfiguration, string, error) {
		return performContainerTransform(adapter, config, portAlloc)
	}
	if isCNBImage(config) {
		return updateForCNBImage(adapter, config, next)
	}
	return updateForShDashC(adapter, config, next)
}

func updateForShDashC(adapter types.ContainerAdapter, ic ImageConfiguration, transformer func(types.ContainerAdapter, ImageConfiguration) (types.ContainerDebugConfiguration, string, error)) (types.ContainerDebugConfiguration, string, error) {
	var rewriter func([]string)
	copy := ic
	switch {
	// Case 1: entrypoint = ["/bin/sh", "-c"], arguments = ["<cmd-line>", args ...]
	case len(ic.Entrypoint) == 2 && len(ic.Arguments) > 0 && isShDashC(ic.Entrypoint[0], ic.Entrypoint[1]):
		if split, err := shell.Split(ic.Arguments[0]); err == nil {
			copy.Entrypoint = split
			copy.Arguments = nil
			rewriter = func(rewrite []string) {
				container := adapter.GetContainer()
				container.Command = nil // inherit from container
				container.Args = append([]string{shJoin(rewrite)}, ic.Arguments[1:]...)
			}
		}

	// Case 2: entrypoint = ["/bin/sh", "-c", "<cmd-line>", args...], arguments = [args ...]
	case len(ic.Entrypoint) > 2 && isShDashC(ic.Entrypoint[0], ic.Entrypoint[1]):
		if split, err := shell.Split(ic.Entrypoint[2]); err == nil {
			copy.Entrypoint = split
			copy.Arguments = nil
			rewriter = func(rewrite []string) {
				container := adapter.GetContainer()
				container.Command = append([]string{ic.Entrypoint[0], ic.Entrypoint[1], shJoin(rewrite)}, ic.Entrypoint[3:]...)
			}
		}

	// Case 3: entrypoint = [] or an entrypoint launcher (and so ignored), arguments = ["/bin/sh", "-c", "<cmd-line>", args...]
	case (len(ic.Entrypoint) == 0 || isEntrypointLauncher(ic.Entrypoint)) && len(ic.Arguments) > 2 && isShDashC(ic.Arguments[0], ic.Arguments[1]):
		if split, err := shell.Split(ic.Arguments[2]); err == nil {
			copy.Entrypoint = split
			copy.Arguments = nil
			rewriter = func(rewrite []string) {
				container := adapter.GetContainer()
				container.Command = nil
				container.Args = append([]string{ic.Arguments[0], ic.Arguments[1], shJoin(rewrite)}, ic.Arguments[3:]...)
			}
		}
	}

	c, image, err := transformer(adapter, copy)
	container := adapter.GetContainer()
	if err == nil && rewriter != nil && container.Command != nil {
		rewriter(container.Command)
	}
	return c, image, err
}

func isShDashC(cmd, arg string) bool {
	return (cmd == "/bin/sh" || cmd == "/bin/bash") && arg == "-c"
}

func performContainerTransform(adapter types.ContainerAdapter, config ImageConfiguration, portAlloc PortAllocator) (types.ContainerDebugConfiguration, string, error) {
	log.Entry(context.TODO()).Tracef("Examining container %q with config %v", adapter.GetContainer().Name, config)
	for _, transform := range containerTransforms {
		if transform.IsApplicable(config) {
			return transform.Apply(adapter, config, portAlloc, Protocols)
		}
	}
	return types.ContainerDebugConfiguration{}, "", fmt.Errorf("unable to determine runtime for %q", adapter.GetContainer().Name)
}
