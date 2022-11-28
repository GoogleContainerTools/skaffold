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

import latestV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest/v2"

// SkaffoldConfigSet encapsulates a slice of skaffold configurations.
type SkaffoldConfigSet []*SkaffoldConfigEntry

// SkaffoldConfigEntry encapsulates a single skaffold configuration, along with the source filename and its index in that file.
type SkaffoldConfigEntry struct {
	*latestV2.SkaffoldConfig
	SourceFile   string
	SourceIndex  int
	IsRootConfig bool
	IsRemote     bool
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
