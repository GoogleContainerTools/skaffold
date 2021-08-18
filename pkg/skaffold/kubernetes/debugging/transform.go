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

package debugging

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging/adapter"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

var (
	decodeFromYaml = scheme.Codecs.UniversalDeserializer().Decode
	encodeAsYaml   = func(o runtime.Object) ([]byte, error) {
		s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		if err := s.Encode(o, w); err != nil {
			return nil, err
		}
		w.Flush()
		return b.Bytes(), nil
	}
)

// ApplyDebuggingTransforms applies language-platform-specific transforms to a list of manifests.
func ApplyDebuggingTransforms(l manifest.ManifestList, builds []graph.Artifact, registries manifest.Registries) (manifest.ManifestList, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	retriever := func(image string) (debug.ImageConfiguration, error) {
		return debug.ConfigRetriever(ctx, image, builds, registries.InsecureRegistries)
	}

	return applyDebuggingTransforms(l, retriever, registries.DebugHelpersRegistry)
}

func applyDebuggingTransforms(l manifest.ManifestList, retriever debug.ConfigurationRetriever, debugHelpersRegistry string) (manifest.ManifestList, error) {
	var updated manifest.ManifestList
	for _, manifest := range l {
		obj, _, err := decodeFromYaml(manifest, nil, nil)
		if err != nil {
			log.Entry(context.Background()).Debugf("Unable to interpret manifest for debugging: %v\n", err)
		} else if transformManifest(obj, retriever, debugHelpersRegistry) {
			manifest, err = encodeAsYaml(obj)
			if err != nil {
				return nil, fmt.Errorf("marshalling yaml: %w", err)
			}
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				log.Entry(context.Background()).Debugln("Applied debugging transform:\n", string(manifest))
			}
		}
		updated = append(updated, manifest)
	}

	return updated, nil
}

func describe(obj runtime.Object) (group, version, kind, description string) {
	// get metadata/name; shamelessly stolen from from k8s.io/cli-runtime/pkg/printers/name.go
	name := "<unknown>"
	if acc, err := meta.Accessor(obj); err == nil {
		if n := acc.GetName(); len(n) > 0 {
			name = n
		}
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	group = gvk.Group
	version = gvk.Version
	kind = gvk.Kind
	if group == "" {
		description = fmt.Sprintf("%s/%s", strings.ToLower(kind), name)
	} else {
		description = fmt.Sprintf("%s.%s/%s", strings.ToLower(kind), group, name)
	}
	return
}

// allocatePort walks the podSpec's containers looking for an available port that is close to desiredPort.
// We deal with wrapping and avoid allocating ports < 1024
func allocatePort(podSpec *v1.PodSpec, desiredPort int32) int32 {
	var maxPort int32 = 65535 // ports are normally [1-65535]
	if desiredPort < 1024 || desiredPort > maxPort {
		desiredPort = 1024 // skip reserved ports
	}
	// We assume ports are rather sparsely allocated, so even if desiredPort
	// is allocated, desiredPort+1 or desiredPort+2 are likely to be free
	for port := desiredPort; port < maxPort; port++ {
		if isPortAvailable(podSpec, port) {
			return port
		}
	}
	for port := desiredPort; port > 1024; port-- {
		if isPortAvailable(podSpec, port) {
			return port
		}
	}
	panic("cannot find available port") // exceedingly unlikely
}

// isPortAvailable returns true if none of the pod's containers specify the given port.
func isPortAvailable(podSpec *v1.PodSpec, port int32) bool {
	for _, container := range podSpec.Containers {
		for _, portSpec := range container.Ports {
			if portSpec.ContainerPort == port {
				return false
			}
		}
	}
	return true
}

// rewriteProbes rewrites k8s probes to expand timeouts to 10 minutes to allow debugging local probes.
func rewriteProbes(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec) bool {
	var minTimeout time.Duration = 10 * time.Minute // make it configurable?
	if annotation, found := metadata.Annotations[types.DebugProbeTimeouts]; found {
		if annotation == "skip" {
			log.Entry(context.Background()).Debugf("skipping probe rewrite on %q by request", metadata.Name)
			return false
		}
		if d, err := time.ParseDuration(annotation); err != nil {
			log.Entry(context.Background()).Warnf("invalid probe timeout value for %q: %q: %v", metadata.Name, annotation, err)
		} else {
			minTimeout = d
		}
	}
	annotation, found := metadata.Annotations[types.DebugConfig]
	if !found {
		log.Entry(context.Background()).Debugf("skipping probe rewrite on %q: not configured for debugging", metadata.Name)
		return false
	}
	var config map[string]types.ContainerDebugConfiguration
	if err := json.Unmarshal([]byte(annotation), &config); err != nil {
		log.Entry(context.Background()).Warnf("error unmarshalling debugging configuration for %q: %v", metadata.Name, err)
		return false
	}

	changed := false
	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		// only affect containers listed in debug-config
		if _, found := config[c.Name]; found {
			lp := rewriteHTTPGetProbe(c.LivenessProbe, minTimeout)
			rp := rewriteHTTPGetProbe(c.ReadinessProbe, minTimeout)
			sp := rewriteHTTPGetProbe(c.StartupProbe, minTimeout)
			if lp || rp || sp {
				log.Entry(context.Background()).Infof("Updated probe timeouts for %s/%s", metadata.Name, c.Name)
			}
			changed = changed || lp || rp || sp
		}
	}
	return changed
}

func rewriteHTTPGetProbe(probe *v1.Probe, minTimeout time.Duration) bool {
	if probe == nil || probe.HTTPGet == nil || int32(minTimeout.Seconds()) < probe.TimeoutSeconds {
		return false
	}
	probe.TimeoutSeconds = int32(minTimeout.Seconds())
	return true
}

// transformManifest attempts to configure a manifest for debugging.
// Returns true if changed, false otherwise.
func transformManifest(obj runtime.Object, retrieveImageConfiguration debug.ConfigurationRetriever, debugHelpersRegistry string) bool {
	one := int32(1)
	switch o := obj.(type) {
	case *v1.Pod:
		return transformPodSpec(&o.ObjectMeta, &o.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *v1.PodList:
		changed := false
		for i := range o.Items {
			if transformPodSpec(&o.Items[i].ObjectMeta, &o.Items[i].Spec, retrieveImageConfiguration, debugHelpersRegistry) {
				changed = true
			}
		}
		return changed
	case *v1.ReplicationController:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.Deployment:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.DaemonSet:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.ReplicaSet:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *appsv1.StatefulSet:
		if o.Spec.Replicas != nil {
			o.Spec.Replicas = &one
		}
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)
	case *batchv1.Job:
		return transformPodSpec(&o.Spec.Template.ObjectMeta, &o.Spec.Template.Spec, retrieveImageConfiguration, debugHelpersRegistry)

	default:
		group, version, _, description := describe(obj)
		if group == "apps" || group == "batch" {
			if version != "v1" {
				// treat deprecated objects as errors
				log.Entry(context.Background()).Errorf("deprecated versions not supported by debug: %s (%s)", description, version)
			} else {
				log.Entry(context.Background()).Warnf("no debug transformation for: %s", description)
			}
		} else {
			log.Entry(context.Background()).Debugf("no debug transformation for: %s", description)
		}
		return false
	}
}

// transformPodSpec attempts to configure a podspec for debugging.
// Returns true if changed, false otherwise.
func transformPodSpec(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration debug.ConfigurationRetriever, debugHelpersRegistry string) bool {
	// order matters as rewriteProbes only affects containers marked for debugging
	containers := rewriteContainers(metadata, podSpec, retrieveImageConfiguration, debugHelpersRegistry)
	timeouts := rewriteProbes(metadata, podSpec)
	return containers || timeouts
}

func rewriteContainers(metadata *metav1.ObjectMeta, podSpec *v1.PodSpec, retrieveImageConfiguration debug.ConfigurationRetriever, debugHelpersRegistry string) bool {
	// skip annotated podspecs — allows users to customize their own image
	if _, found := metadata.Annotations[types.DebugConfig]; found {
		return false
	}

	portAlloc := func(desiredPort int32) int32 {
		return allocatePort(podSpec, desiredPort)
	}
	// map of containers -> debugging configuration maps; k8s ensures that a pod's containers are uniquely named
	configurations := make(map[string]types.ContainerDebugConfiguration)
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
		if configuration, requiredImage, err := debug.TransformContainer(a, imageConfig, portAlloc); err == nil {
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
		supportVolume := v1.Volume{Name: debug.DebuggingSupportFilesVolume, VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}}
		podSpec.Volumes = append(podSpec.Volumes, supportVolume)

		// this volume is mounted in the containers at `/dbg`
		supportVolumeMount := v1.VolumeMount{Name: debug.DebuggingSupportFilesVolume, MountPath: "/dbg"}
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
		metadata.Annotations[types.DebugConfig] = debug.EncodeConfigurations(configurations)
		return true
	}
	return false
}
