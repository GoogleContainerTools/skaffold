/*
Copyright 2019 The Skaffold Authors

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

package runner

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
)

type changes struct {
	dirtyArtifacts []*artifactChange
	needsRebuild   []*latest.Artifact
	needsResync    []*sync.Item
	needsRedeploy  bool
	needsReload    bool
}

type artifactChange struct {
	artifact *latest.Artifact
	events   watch.Events
}

func (c *changes) AddDirtyArtifact(a *latest.Artifact, e watch.Events) {
	c.dirtyArtifacts = append(c.dirtyArtifacts, &artifactChange{artifact: a, events: e})
}

func (c *changes) AddRebuild(a *latest.Artifact) {
	c.needsRebuild = append(c.needsRebuild, a)
}

func (c *changes) AddResync(s *sync.Item) {
	c.needsResync = append(c.needsResync, s)
}

func (c *changes) reset() {
	c.dirtyArtifacts = nil
	c.needsRebuild = nil
	c.needsResync = nil

	c.needsRedeploy = false
	c.needsReload = false
}
