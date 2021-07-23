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
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
)

type Container struct {
	Name string
	Id   string
}

type ContainerTracker struct {
	sync.RWMutex
	deployedContainers  map[string]Container      // artifact image name -> container
	containerToArtifact map[string]graph.Artifact // container id -> artifact (for colorpicker)
	containers          map[string]bool           // set of tracked container ids
	notifier            chan string
}

// NewContainerTracker creates a new ContainerTracker.
func NewContainerTracker() *ContainerTracker {
	return &ContainerTracker{
		containerToArtifact: make(map[string]graph.Artifact),
		deployedContainers:  make(map[string]Container),
		notifier:            make(chan string, 1),
	}
}

// Notifier returns the notifier channel for this tracker.
// Used by the log streamer to be notified when a new container is
// added to the tracker.
func (t *ContainerTracker) Notifier() chan string {
	return t.notifier
}

func (t *ContainerTracker) ArtifactForContainer(id string) graph.Artifact {
	t.Lock()
	defer t.Unlock()
	return t.containerToArtifact[id]
}

// ContainerIdForImage returns the deployed container created from
// the provided image, if it exists.
func (t *ContainerTracker) ContainerForImage(image string) (Container, bool) {
	t.Lock()
	defer t.Unlock()
	c, found := t.deployedContainers[image]
	return c, found
}

// DeployedContainers returns the list of all containers deployed
// during this run.
func (t *ContainerTracker) DeployedContainers() map[string]Container {
	return t.deployedContainers
}

// Reset resets the entire tracker when a new dev cycle starts.
func (t *ContainerTracker) Reset() {
	for c := range t.containers {
		delete(t.containers, c)
	}
}

// Add adds a container to the list.
func (t *ContainerTracker) Add(artifact graph.Artifact, c Container) {
	t.Lock()
	defer t.Unlock()
	t.deployedContainers[artifact.ImageName] = c
	t.containerToArtifact[c.Id] = artifact
	go func() {
		t.notifier <- c.Id
	}()
}
