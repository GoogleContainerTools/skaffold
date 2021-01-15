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

package docker

import (
	"sync"
)

type ContainerTracker struct {
	sync.RWMutex
	containers       map[string]bool
	imageToContainer map[string]string
	notifier         chan string

	// TODO(nkubala): these should probably live on the Logger
	stoppers map[string]chan bool
}

// NewContainerTracker creates a new ContainerTracker.
func NewContainerTracker() *ContainerTracker {
	return &ContainerTracker{
		containers:       make(map[string]bool),
		imageToContainer: make(map[string]string),
		notifier:         make(chan string, 1),
		stoppers:         make(map[string]chan bool),
	}
}

func (t *ContainerTracker) Reset() {
	for c, _ := range t.containers {
		t.stoppers[c] <- true
		delete(t.containers, c)
	}
}

// Add adds a container to the list.
func (t *ContainerTracker) Add(image, id string) {
	t.Lock()
	t.containers[id] = true
	t.imageToContainer[image] = id
	t.notifier <- id
	t.stoppers[id] = make(chan bool, 1)
	t.Unlock()
}

// TODO(nkubala): implement?
// func (t *ContainerTracker) Remove(image, id string) {
// }
