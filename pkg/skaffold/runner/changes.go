/*
Copyright 2018 The Skaffold Authors

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
	latest "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
)

type changes struct {
	dirtyArtifacts []*latest.Artifact
	needsRedeploy  bool
	needsReload    bool
}

func (c *changes) Add(a *latest.Artifact) {
	c.dirtyArtifacts = append(c.dirtyArtifacts, a)
}

func (c *changes) reset() {
	c.dirtyArtifacts = nil
	c.needsRedeploy = false
	c.needsReload = false
}
