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

	batchv1 "k8s.io/api/batch/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
)

type Job struct {
	Name string
	ID   string
}

type JobTracker struct {
	sync.RWMutex
	deployedContainers  map[string]Job            // artifact image name -> container
	containerToArtifact map[string]graph.Artifact // container id -> artifact (for colorpicker)
	deployedJobs        map[string]*batchv1.Job   // job.Name -> batchv1.Job (for cleanup)
	containers          map[string]bool           // set of tracked container ids
	notifier            chan []string
}

// NewContainerTracker creates a new ContainerTracker.
func NewContainerTracker() *JobTracker {
	return &JobTracker{
		containerToArtifact: make(map[string]graph.Artifact),
		deployedContainers:  make(map[string]Job),
		deployedJobs:        make(map[string]*batchv1.Job),
		notifier:            make(chan []string, 1),
	}
}

// Notifier returns the notifier channel for this tracker.
// Used by the log streamer to be notified when a new container is
// added to the tracker.
func (t *JobTracker) Notifier() chan []string {
	return t.notifier
}

func (t *JobTracker) ArtifactForContainer(id string) graph.Artifact {
	t.Lock()
	defer t.Unlock()
	return t.containerToArtifact[id]
}

// ContainerIdForImage returns the deployed container created from
// the provided image, if it exists.
func (t *JobTracker) ContainerForImage(image string) (Job, bool) {
	t.Lock()
	defer t.Unlock()
	c, found := t.deployedContainers[image]
	return c, found
}

// DeployedContainers returns the list of all containers deployed
// during this run.
func (t *JobTracker) DeployedContainers() map[string]Job {
	return t.deployedContainers
}

// Reset resets the entire tracker when a new dev cycle starts.
func (t *JobTracker) Reset() {
	for c := range t.containers {
		delete(t.containers, c)
	}
}

// Add adds a container to the list.
func (t *JobTracker) Add(artifact graph.Artifact, c Job, namespace string) {
	t.Lock()
	defer t.Unlock()
	t.deployedContainers[artifact.ImageName] = c
	t.containerToArtifact[c.ID] = artifact
	go func() {
		t.notifier <- []string{c.ID, namespace}
	}()
}

// Add adds a Job to the list.
func (t *JobTracker) AddJob(job *batchv1.Job) {
	t.Lock()
	defer t.Unlock()
	t.deployedJobs[job.Name] = job
}

// DeployedJobs returns the list of all jobs deployed
// during this run.
func (t *JobTracker) DeployedJobs() map[string]*batchv1.Job {
	return t.deployedJobs
}
