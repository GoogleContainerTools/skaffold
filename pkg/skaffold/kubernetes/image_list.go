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

package kubernetes

import (
	"sync"

	v1 "k8s.io/api/core/v1"
)

// PodSelector is used to choose which pods to log.
type PodSelector interface {
	Select(pod *v1.Pod) bool
}

// ImageList implements PodSelector based on a list of images names.
type ImageList struct {
	sync.RWMutex
	names map[string]bool
}

// NewImageList creates a new ImageList.
func NewImageList() *ImageList {
	return &ImageList{
		names: make(map[string]bool),
	}
}

// Add adds an image to the list.
func (l *ImageList) Add(image string) {
	l.Lock()
	l.names[image] = true
	l.Unlock()
}

// Select returns true if one of the pod's images is in the list.
func (l *ImageList) Select(pod *v1.Pod) bool {
	l.RLock()
	defer l.RUnlock()

	for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		if l.names[container.Image] {
			return true
		}
	}

	return false
}
