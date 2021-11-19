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

package parser

import (
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// SkaffoldConfigSet encapsulates a slice of skaffold configurations.
type SkaffoldConfigSet []*SkaffoldConfigEntry

// SkaffoldConfigEntry encapsulates a single skaffold configuration, along with the source filename and its index in that file.
type SkaffoldConfigEntry struct {
	*latestV1.SkaffoldConfig
	SourceFile   string
	SourceIndex  int
	IsRootConfig bool
	IsRemote     bool
	YAMLNodes    *YAMLNodes
}

// SelectRootConfigs filters SkaffoldConfigSet to only configs read from the root skaffold.yaml file
func (s SkaffoldConfigSet) SelectRootConfigs() SkaffoldConfigSet {
	var filteredSet SkaffoldConfigSet
	for _, entry := range s {
		if entry.IsRootConfig {
			filteredSet = append(filteredSet, entry)
		}
	}
	return filteredSet
}

// Locate gets the location for a skaffold schema struct pointer
func (s SkaffoldConfigSet) Locate(obj interface{}) *Location {
	loc := MissingLocation()
	for _, c := range s {
		if l := c.YAMLNodes.Locate(obj); l.StartLine != -1 {
			loc = l
		}
	}
	return loc
}
