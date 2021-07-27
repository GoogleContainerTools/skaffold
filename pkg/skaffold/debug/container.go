package debug

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/annotations"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	shell "github.com/kballard/go-shellquote"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type operableContainer struct {
	Name    string
	Command []string
	Args    []string
	Env     containerEnv
	Ports   []containerPort
}

// adapted from github.com/kubernetes/api/core/v1/types.go
type containerPort struct {
	Name          string
	HostPort      int32
	ContainerPort int32
	Protocol      string
	HostIP        string
}

type containerEnv struct {
	Order []string
	Env   map[string]string
}

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
	Apply(container *operableContainer, config imageConfiguration, portAlloc portAllocator, overrideProtocols []string) (annotations.ContainerDebugConfiguration, string, error)
}

// transformContainer rewrites the container definition to enable debugging.
// Returns a debugging configuration description with associated language runtime support
// container image, or an error if the rewrite was unsuccessful.
func transformContainer(container *operableContainer, config imageConfiguration, portAlloc portAllocator) (annotations.ContainerDebugConfiguration, string, error) {
	// Update the image configuration's environment with those set in the k8s manifest.
	// (Environment variables in the k8s container's `env` add to the image configuration's `env` settings rather than replace.)
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
	next := func(container *operableContainer, config imageConfiguration) (annotations.ContainerDebugConfiguration, string, error) {
		return performContainerTransform(container, config, portAlloc)
	}
	if isCNBImage(config) {
		return updateForCNBImage(container, config, next)
	}
	return updateForShDashC(container, config, next)
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
		operable := operableContainerFromK8sContainer(&container)
		// requiredImage, if not empty, is the image ID providing the debugging support files
		// `err != nil` means that the container did not or could not be transformed
		if configuration, requiredImage, err := transformContainer(operable, imageConfig, portAlloc); err == nil {
			configuration.Artifact = imageConfig.artifact
			if configuration.WorkingDir == "" {
				configuration.WorkingDir = imageConfig.workingDir
			}
			configurations[container.Name] = configuration
			applyFromOperable(operable, &container)
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

func updateForShDashC(container *operableContainer, ic imageConfiguration, transformer func(*operableContainer, imageConfiguration) (annotations.ContainerDebugConfiguration, string, error)) (annotations.ContainerDebugConfiguration, string, error) {
	var rewriter func([]string)
	copy := ic
	switch {
	// Case 1: entrypoint = ["/bin/sh", "-c"], arguments = ["<cmd-line>", args ...]
	case len(ic.entrypoint) == 2 && len(ic.arguments) > 0 && isShDashC(ic.entrypoint[0], ic.entrypoint[1]):
		if split, err := shell.Split(ic.arguments[0]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
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
				container.Command = append([]string{ic.entrypoint[0], ic.entrypoint[1], shJoin(rewrite)}, ic.entrypoint[3:]...)
			}
		}

	// Case 3: entrypoint = [] or an entrypoint launcher (and so ignored), arguments = ["/bin/sh", "-c", "<cmd-line>", args...]
	case (len(ic.entrypoint) == 0 || isEntrypointLauncher(ic.entrypoint)) && len(ic.arguments) > 2 && isShDashC(ic.arguments[0], ic.arguments[1]):
		if split, err := shell.Split(ic.arguments[2]); err == nil {
			copy.entrypoint = split
			copy.arguments = nil
			rewriter = func(rewrite []string) {
				container.Command = nil
				container.Args = append([]string{ic.arguments[0], ic.arguments[1], shJoin(rewrite)}, ic.arguments[3:]...)
			}
		}
	}

	c, image, err := transformer(container, copy)
	if err == nil && rewriter != nil && container.Command != nil {
		rewriter(container.Command)
	}
	return c, image, err
}

func isShDashC(cmd, arg string) bool {
	return (cmd == "/bin/sh" || cmd == "/bin/bash") && arg == "-c"
}

func performContainerTransform(container *operableContainer, config imageConfiguration, portAlloc portAllocator) (annotations.ContainerDebugConfiguration, string, error) {
	log.Entry(context.Background()).Tracef("Examining container %q with config %v", container.Name, config)
	for _, transform := range containerTransforms {
		if transform.IsApplicable(config) {
			return transform.Apply(container, config, portAlloc, Protocols)
		}
	}
	return annotations.ContainerDebugConfiguration{}, "", fmt.Errorf("unable to determine runtime for %q", container.Name)
}

// operableContainerFromK8sContainer creates an instance of an operableContainer
// from a v1.Container reference. This object will be passed around to accept
// transforms, and will eventually overwrite fields from the creating v1.Container
// in the manifest-under-transformation's pod spec.
func operableContainerFromK8sContainer(c *v1.Container) *operableContainer {
	return &operableContainer{
		Command: c.Command,
		Args:    c.Args,
		Env:     k8sEnvToContainerEnv(c.Env),
		Ports:   k8sPortsToContainerPorts(c.Ports),
	}
}

func k8sEnvToContainerEnv(k8sEnv []v1.EnvVar) containerEnv {
	// TODO(nkubala): ValueFrom is ignored. Do we care?
	env := make(map[string]string, len(k8sEnv))
	var order []string
	for _, entry := range k8sEnv {
		order = append(order, entry.Name)
		env[entry.Name] = entry.Value
	}
	return containerEnv{
		Order: order,
		Env:   env,
	}
}

func containerEnvToK8sEnv(env containerEnv) []v1.EnvVar {
	var k8sEnv []v1.EnvVar
	for _, k := range env.Order {
		k8sEnv = append(k8sEnv, v1.EnvVar{
			Name:  k,
			Value: env.Env[k],
		})
	}
	return k8sEnv
}

func k8sPortsToContainerPorts(k8sPorts []v1.ContainerPort) []containerPort {
	var containerPorts []containerPort
	for _, port := range k8sPorts {
		containerPorts = append(containerPorts, containerPort{
			Name:          port.Name,
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			Protocol:      string(port.Protocol),
			HostIP:        port.HostIP,
		})
	}
	return containerPorts
}

func containerPortsToK8sPorts(containerPorts []containerPort) []v1.ContainerPort {
	var k8sPorts []v1.ContainerPort
	for _, port := range containerPorts {
		k8sPorts = append(k8sPorts, v1.ContainerPort{
			Name:          port.Name,
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			Protocol:      v1.Protocol(port.Protocol),
			HostIP:        port.HostIP,
		})
	}
	return k8sPorts
}

// applyFromOperable takes the relevant fields from the operable container
// and applies them to the referenced v1.Container from the manifest's pod spec
func applyFromOperable(o *operableContainer, c *v1.Container) {
	c.Args = o.Args
	c.Command = o.Command
	c.Env = containerEnvToK8sEnv(o.Env)
	c.Ports = containerPortsToK8sPorts(o.Ports)
}
