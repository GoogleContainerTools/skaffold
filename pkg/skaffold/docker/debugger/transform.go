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

package debugger

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	SupportVolumeMount = volume.VolumeCreateBody{Name: debug.DebuggingSupportFilesVolume}
	TransformImage     = transformImage // For testing
)

type Transform func(string) string

func transformImage(ctx context.Context, artifact graph.Artifact, cfg *container.Config, insecureRegistries map[string]bool, debugHelpersRegistry string) (map[string]types.ContainerDebugConfiguration, []*container.Config, error) {
	portAvailable := func(port int32) bool {
		return isPortAvailable(cfg, port)
	}
	portAlloc := func(desiredPort int32) int32 {
		return util.AllocatePort(portAvailable, desiredPort)
	}

	adapter := NewAdapter(cfg)
	imageConfig, err := debug.RetrieveImageConfiguration(ctx, &artifact, insecureRegistries)
	if err != nil {
		return nil, nil, err
	}

	configurations := make(map[string]types.ContainerDebugConfiguration)
	var initContainers []*container.Config
	// the container images that require debugging support files
	var containersRequiringSupport []*container.Config
	// the set of image IDs required to provide debugging support files
	requiredSupportImages := make(map[string]bool)

	if configuration, requiredImage, err := debug.TransformContainer(adapter, imageConfig, portAlloc); err == nil {
		configurations[adapter.GetContainer().Name] = configuration
		if len(requiredImage) > 0 {
			log.Entry(ctx).Infof("%q requires debugging support image %q", cfg.Image, requiredImage)
			containersRequiringSupport = append(containersRequiringSupport, cfg)
			requiredSupportImages[requiredImage] = true
		}
	} else {
		log.Entry(ctx).Warnf("Image %q not configured for debugging: %v", cfg.Image, err)
	}

	// check if we have any images requiring additional debugging support files
	if len(containersRequiringSupport) > 0 {
		log.Entry(ctx).Infof("Configuring installation of debugging support files")
		// the initContainers are responsible for populating the contents of `/dbg`
		for imageID := range requiredSupportImages {
			supportFilesInitContainer := &container.Config{
				Image:   fmt.Sprintf("%s/%s", debugHelpersRegistry, imageID),
				Volumes: map[string]struct{}{"/dbg": {}},
			}
			initContainers = append(initContainers, supportFilesInitContainer)
		}
		// the populated volume is then mounted in the containers at `/dbg` too
		for _, container := range containersRequiringSupport {
			if container.Volumes == nil {
				container.Volumes = make(map[string]struct{})
			}
			container.Volumes["/dbg"] = struct{}{}
		}
	}

	return configurations, initContainers, nil
}

// isPortAvailable returns true if none of the pod's containers specify the given port.
func isPortAvailable(cfg *container.Config, port int32) bool {
	for exposedPort := range cfg.ExposedPorts {
		if int32(exposedPort.Int()) == port {
			return false
		}
	}
	return true
}
