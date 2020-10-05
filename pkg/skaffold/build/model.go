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

package build

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// buildNode models the artifact dependency graph using a set of channels.
// Each build node has a wait  channel which it closes once it completes building by calling markComplete.
// This notifies all listeners waiting for this node of a successful build.
// Additionally it has a reference to the channels for each of its dependencies.
// Calling `waitForDependencies` ensures that all required nodes' channels have already been closed and as such have finished building before the current artifact build starts.
type buildNode struct {
	imageName    string
	wait         chan interface{}
	dependencies []buildNode
}

// markComplete broadcasts a successful build
func (a *buildNode) markComplete() {
	// closing channel notifies all listeners waiting for this build that it succeeded
	close(a.wait)
}

// waitForDependencies returns an error if any dependency build fails
func (a *buildNode) waitForDependencies(ctx context.Context) error {
	for _, dep := range a.dependencies {
		// wait for required builds to complete
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-dep.wait:
		}
	}
	return nil
}

func createNodes(artifacts []*latest.Artifact) []buildNode {
	nodeMap := make(map[string]buildNode)
	for _, a := range artifacts {
		nodeMap[a.ImageName] = buildNode{
			imageName: a.ImageName,
			wait:      make(chan interface{}),
		}
	}

	var nodes []buildNode
	for _, a := range artifacts {
		ar := nodeMap[a.ImageName]
		for _, d := range a.Dependencies {
			ch, found := nodeMap[d.ImageName]
			if !found {
				continue
			}
			ar.dependencies = append(ar.dependencies, ch)
		}
		nodes = append(nodes, ar)
	}
	return nodes
}

// countingSemaphore uses a buffered channel of size `n` that acts like a counting semaphore, allowing up to `n` concurrent operations
type countingSemaphore struct {
	sem chan bool
}

func newCountingSemaphore(count int) countingSemaphore {
	return countingSemaphore{sem: make(chan bool, count)}
}

func (c countingSemaphore) acquire() (release func()) {
	c.sem <- true
	return func() {
		<-c.sem
	}
}
