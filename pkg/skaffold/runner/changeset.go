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
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
)

type ChangeSet struct {
	needsRebuild   []*latest_v1.Artifact
	rebuildTracker map[string]*latest_v1.Artifact
	needsResync    []*sync.Item
	resyncTracker  map[string]*sync.Item
	needsRetest    map[string]bool // keyed on artifact image name
	needsRedeploy  bool
	needsReload    bool
}

func (c *ChangeSet) NeedsRebuild() []*latest_v1.Artifact {
	return c.needsRebuild
}

func (c *ChangeSet) NeedsResync() []*sync.Item {
	return c.needsResync
}

func (c *ChangeSet) NeedsRedeploy() bool {
	return c.needsRedeploy
}

func (c *ChangeSet) NeedsRetest() map[string]bool {
	return c.needsRetest
}

func (c *ChangeSet) NeedsReload() bool {
	return c.needsReload
}

func (c *ChangeSet) AddRebuild(a *latest_v1.Artifact) {
	if _, ok := c.rebuildTracker[a.ImageName]; ok {
		return
	}

	if c.rebuildTracker == nil {
		c.rebuildTracker = map[string]*latest_v1.Artifact{}
	}
	c.rebuildTracker[a.ImageName] = a
	c.needsRebuild = append(c.needsRebuild, a)
}

func (c *ChangeSet) AddRetest(a *latest_v1.Artifact) {
	if c.needsRetest == nil {
		c.needsRetest = make(map[string]bool)
	}
	c.needsRetest[a.ImageName] = true
}

func (c *ChangeSet) AddResync(s *sync.Item) {
	if _, ok := c.resyncTracker[s.Image]; ok {
		return
	}

	if c.resyncTracker == nil {
		c.resyncTracker = map[string]*sync.Item{}
	}
	c.resyncTracker[s.Image] = s
	c.needsResync = append(c.needsResync, s)
}

func (c *ChangeSet) ResetBuild() {
	c.rebuildTracker = make(map[string]*latest_v1.Artifact)
	c.needsRebuild = nil
}

func (c *ChangeSet) ResetSync() {
	c.resyncTracker = make(map[string]*sync.Item)
	c.needsResync = nil
}

func (c *ChangeSet) ResetDeploy() {
	c.needsRedeploy = false
}

func (c *ChangeSet) Redeploy() {
	c.needsRedeploy = true
}

func (c *ChangeSet) Reload() {
	c.needsReload = true
}

func (c *ChangeSet) ResetTest() {
	c.needsRetest = make(map[string]bool)
}
