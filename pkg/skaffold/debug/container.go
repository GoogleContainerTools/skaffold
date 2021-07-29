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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/annotations"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging/adapter"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

// containerTransforms are the set of configured transformers
var containerTransforms []containerTransformer

// containerTransformer transforms a container definition
type containerTransformer interface {
	// IsApplicable determines if this container is suitable to be transformed.
	IsApplicable(config imageConfiguration) bool

	// Apply configures a container definition for debugging, returning the debug configuration details
	// and required initContainer (an empty string if not required), or return a non-nil error if
	// the container could not be transformed.  The initContainer image is intended to install any
	// required debug support tools.
	Apply(adapter types.ContainerAdapter, config imageConfiguration, portAlloc portAllocator, overrideProtocols []string) (annotations.ContainerDebugConfiguration, string, error)
}

// transformContainer rewrites the container definition to enable debugging.
// Returns a debugging configuration description with associated language runtime support
// container image, or an error if the rewrite was unsuccessful.
func transformContainer(adapter types.ContainerAdapter, config imageConfiguration, portAlloc portAllocator) (annotations.ContainerDebugConfiguration, string, error) {
	// Update the image configuration's environment with those set in the k8s manifest.
	// (Environment variables in the k8s container's `env` add to the image configuration's `env` settings rather than replace.)
	container := adapter.GetContainer()
	defer adapter.Apply()
	for _, key := range container.Env.Order {
		// FIXME handle ValueFrom?
		if config.env == nil {
			config.env = make(map[string]string)
		}
		config.env[key] = container.Env.Env[key]
	}

	if len(container.Command) > 0 {
		config.entrypoint = container.Command
	}
	if len(container.Args) > 0 {
		config.arguments = container.Args
	}

	// Apply command-line unwrapping for buildpack images and images using `sh -c`-style command-lines
	next := func(adapter types.ContainerAdapter, config imageConfiguration) (annotations.ContainerDebugConfiguration, string, error) {
		return performContainerTransform(adapter, config, portAlloc)
	}
	if isCNBImage(config) {
		return updateForCNBImage(adapter, config, next)
	}
	return updateForShDashC(adapter, config, next)
}

func rewriteContainers(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration configurationRetriever, debugHelpersRegistry string) bool {
	// skip annotated podspecs — allows users to customize their own image
	if _, found := metadata.Annotations[annotations.DebugConfig]; found {
		return false
	}

	portAlloc := func(desiredPort int32) int32 {
		return allocatePort(podSpec, desiredPort)
	}
	// map of containers -> debugging configuration maps; k8s ensures that a pod's containers are uniquely named
	configurations := make(map[string]annotations.ContainerDebugConfiguration)
	// the container images that require debugging support files
	var containersRequiringSupport []*v1.Container
	// the set of image IDs required to provide debugging support files
	requiredSupportImages := make(map[string]bool)
	for i := range podSpec.Containers {
		container := podSpec.Containers[i] // make a copy and only apply changes on successful transform

		// the usual retriever returns an error for non-build artifacts
		imageConfig, err := retrieveImageConfiguration(container.Image)
		if err != nil {
			continue
		}
		a := adapter.NewAdapter(&container)
		// requiredImage, if not empty, is the image ID providing the debugging support files
		// `err != nil` means that the container did not or could not be transformed
		if configuration, requiredImage, err := transformContainer(a, imageConfig, portAlloc); err == nil {
			configuration.Artifact = imageConfig.artifact
			if configuration.WorkingDir == "" {
				configuration.WorkingDir = imageConfig.workingDir
			}
			configurations[container.Name] = configuration
			podSpec.Containers[i] = container // apply any configuration changes
			if len(requiredImage) > 0 {
				log.Entry(context.Background()).Infof("%q requires debugging support image %q", container.Name, requiredImage)
				containersRequiringSupport = append(containersRequiringSupport, &podSpec.Containers[i])
				requiredSupportImages[requiredImage] = true
			}
		} else {
			log.Entry(context.Background()).Warnf("Image %q not configured for debugging: %v", container.Name, err)
		}
	}

	// check if we have any images requiring additional debugging support files
	if len(containersRequiringSupport) > 0 {
		log.Entry(context.Background()).Infof("Configuring installation of debugging support files")
		// we create the volume that will hold the debugging support files
		supportVolume := v1.Volume{Name: debuggingSupportFilesVolume, VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}}
		podSpec.Volumes = append(podSpec.Volumes, supportVolume)

		// this volume is mounted in the containers at `/dbg`
		supportVolumeMount := v1.VolumeMount{Name: debuggingSupportFilesVolume, MountPath: "/dbg"}
		// the initContainers are responsible for populating the contents of `/dbg`
		for imageID := range requiredSupportImages {
			supportFilesInitContainer := v1.Container{
				Name:         fmt.Sprintf("install-%s-debug-support", imageID),
				Image:        fmt.Sprintf("%s/%s", debugHelpersRegistry, imageID),
				VolumeMounts: []v1.VolumeMount{supportVolumeMount},
			}
			podSpec.InitContainers = append(podSpec.InitContainers, supportFilesInitContainer)
		}
		// the populated volume is then mounted in the containers at `/dbg` too
		for _, container := range containersRequiringSupport {
			container.VolumeMounts = append(container.VolumeMounts, supportVolumeMount)
		}
	}

	if len(configurations) > 0 {
		if metadata.Annotations == nil {
			metadata.Annotations = make(map[string]string)
		}
		metadata.Annotations[annotations.DebugConfig] = encodeConfigurations(configurations)
		return true
	}
	return false
}

func updateForShDashC(adapter types.ContainerAdapter, ic imageConfiguration, transformer func(types.ContainerAdapter, imageConfiguration) (annotations.ContainerDebugConfiguration, string, error)) (annotations.ContainerDebugConfiguration, string, error) {
	var rewriter func([]string)
	copy := ic
	switch {
	// Case 1: entrypoint = ["/bin/sh", "-c"], arguments = ["<cmd-line>", args ...]
	case len(ic.entrypoint) == 2 && len(ic.arguments) > 0 && isShDashC(ic.entrypoint[0], ic.entrypoint[1]):
		if split, err := shell.Split(ic.arguments[0]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container := adapter.GetContainer()
				container.Command = nil // inherit from container
				container.Args = append([]string{shJoin(rewrite)}, ic.arguments[1:]...)
			}
		}

	// Case 2: entrypoint = ["/bin/sh", "-c", "<cmd-line>", args...], arguments = [args ...]
	case len(ic.entrypoint) > 2 && isShDashC(ic.entrypoint[0], ic.entrypoint[1]):
		if split, err := shell.Split(ic.entrypoint[2]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container := adapter.GetContainer()
				container.Command = append([]string{ic.entrypoint[0], ic.entrypoint[1], shJoin(rewrite)}, ic.entrypoint[3:]...)
			}
		}

	// Case 3: entrypoint = [] or an entrypoint launcher (and so ignored), arguments = ["/bin/sh", "-c", "<cmd-line>", args...]
	case (len(ic.entrypoint) == 0 || isEntrypointLauncher(ic.entrypoint)) && len(ic.arguments) > 2 && isShDashC(ic.arguments[0], ic.arguments[1]):
		if split, err := shell.Split(ic.arguments[2]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container := adapter.GetContainer()
				container.Command = nil
				container.Args = append([]string{ic.arguments[0], ic.arguments[1], shJoin(rewrite)}, ic.arguments[3:]...)
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

func performContainerTransform(adapter types.ContainerAdapter, config imageConfiguration, portAlloc portAllocator) (annotations.ContainerDebugConfiguration, string, error) {
	log.Entry(context.Background()).Tracef("Examining container %q with config %v", adapter.GetContainer().Name, config)
	for _, transform := range containerTransforms {
		if transform.IsApplicable(config) {
			return transform.Apply(adapter, config, portAlloc, Protocols)
		}
	}
	return annotations.ContainerDebugConfiguration{}, "", fmt.Errorf("unable to determine runtime for %q", adapter.GetContainer().Name)
}
