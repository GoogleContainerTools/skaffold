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

package tracker

import (
	"sort"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
)

type ContainerTracker struct {
	sync.RWMutex
	deployedContainers  map[string]string         // image name -> container id
	containers          map[string]bool           // set of tracked container ids
	containerToArtifact map[string]graph.Artifact // container id -> graph.Artifact
	notifier            chan string
}

// NewContainerTracker creates a new ContainerTracker.
func NewContainerTracker() *ContainerTracker {
	return &ContainerTracker{
		deployedContainers:  make(map[string]string),
		containers:          make(map[string]bool),
		containerToArtifact: make(map[string]graph.Artifact),
		notifier:            make(chan string, 1),
	}
}

// Notifier returns the notifier channel for this tracker.
// Used by the log streamer to be notified when a new container is
// added to the tracker.
func (t *ContainerTracker) Notifier() chan string {
	return t.notifier
}

// ImageForContainer maps a container id to the image tag it was created from.
// Used by the ColorPicker to maintain consistency between images and colors.
func (t *ContainerTracker) ArtifactForContainer(id string) graph.Artifact {
	return t.containerToArtifact[id]
}

// DeployedContainerForImage returns a the ID of a deployed container created from
// the provided image, if it exists.
func (t *ContainerTracker) DeployedContainerForImage(image string) string {
	t.Lock()
	defer t.Unlock()
	if id, found := t.deployedContainers[image]; found {
		return id
	}
	return ""
}

// DeployedContainers returns the list of all containers deployed
// during this run.
func (t *ContainerTracker) DeployedContainers() []string {
	var containers []string
	for _, id := range t.deployedContainers {
		containers = append(containers, id)
	}
	sort.Strings(containers)
	return containers
}

// Reset resets the entire tracker when a new dev cycle starts.
func (t *ContainerTracker) Reset() {
	for c := range t.containers {
		delete(t.containers, c)
	}
}

// Add adds a container to the list.
func (t *ContainerTracker) Add(artifact graph.Artifact, id string) {
	t.Lock()
	defer t.Unlock()
	t.containers[id] = true
	t.deployedContainers[artifact.ImageName] = id
	t.containerToArtifact[id] = artifact
	go func() {
		t.notifier <- id
	}()
}
