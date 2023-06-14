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
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
)

var (
	// For testing
	notifyDebuggingContainerStarted    = event.DebuggingContainerStarted
	notifyDebuggingContainerTerminated = event.DebuggingContainerTerminated
	debuggingContainerStartedV2        = eventV2.DebuggingContainerStarted
	debuggingContainerTerminatedV2     = eventV2.DebuggingContainerTerminated
)

type DebugManager struct {
	insecureRegistries   map[string]bool
	debugHelpersRegistry string

	images         []string
	configurations map[string]types.ContainerDebugConfiguration

	supportMounts map[string]mount.Mount
	mountLock     sync.Mutex
}

func NewDebugManager(insecureRegistries map[string]bool, debugHelpersRegistry string) *DebugManager {
	return &DebugManager{
		insecureRegistries:   insecureRegistries,
		debugHelpersRegistry: debugHelpersRegistry,
		configurations:       make(map[string]types.ContainerDebugConfiguration),
		supportMounts:        make(map[string]mount.Mount),
	}
}

func (d *DebugManager) ConfigurationForImage(image string) types.ContainerDebugConfiguration {
	if d == nil {
		return types.ContainerDebugConfiguration{}
	}
	return d.configurations[image]
}

func (d *DebugManager) AddSupportMount(image string, m mount.Mount) {
	d.mountLock.Lock()
	defer d.mountLock.Unlock()
	d.supportMounts[image] = m
}

func (d *DebugManager) HasMount(image string) bool {
	d.mountLock.Lock()
	defer d.mountLock.Unlock()
	_, ok := d.supportMounts[image]
	return ok
}

func (d *DebugManager) SupportMounts() map[string]mount.Mount {
	if d == nil {
		return nil
	}
	return d.supportMounts
}

func (d *DebugManager) Start(context.Context) error {
	if d == nil {
		return nil
	}
	for _, image := range d.images {
		config := d.configurations[image]
		notifyDebuggingContainerStarted(
			"", // no pod
			image,
			"", // no namespace
			config.Artifact,
			config.Runtime,
			config.WorkingDir,
			config.Ports)
		debuggingContainerStartedV2("", image, "", config.Artifact, config.Runtime, config.WorkingDir, config.Ports)
	}
	return nil
}

func (d *DebugManager) Stop() {
	if d == nil {
		return
	}
	for _, image := range d.images {
		config := d.configurations[image]
		notifyDebuggingContainerTerminated(
			"", // no pod
			image,
			"", // no namespace
			config.Artifact,
			config.Runtime,
			config.WorkingDir,
			config.Ports)
		debuggingContainerTerminatedV2("", image, "", config.Artifact, config.Runtime, config.WorkingDir, config.Ports)
	}
	d.images = nil
	d.configurations = make(map[string]types.ContainerDebugConfiguration)
}

func (d *DebugManager) Name() string { return "Docker Debug Manager" }

func (d *DebugManager) TransformImage(ctx context.Context, artifact graph.Artifact, cfg *container.Config) ([]*container.Config, error) {
	if d == nil {
		return nil, nil
	}
	configurations, initContainers, err := TransformImage(ctx, artifact, cfg, d.insecureRegistries, d.debugHelpersRegistry)
	if err != nil {
		return nil, err
	}
	d.images = append(d.images, cfg.Image)
	for k, v := range configurations {
		d.configurations[k] = v
	}

	return initContainers, nil
}
