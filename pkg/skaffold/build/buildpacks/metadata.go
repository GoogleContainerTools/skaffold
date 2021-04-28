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

package buildpacks

import (
	"encoding/json"

	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

type buildMetadata struct {
	Bom []bom `json:"bom"`
}

type bom struct {
	Metadata bomMetadata `json:"metadata"`
}

type bomMetadata struct {
	Sync []syncRule `json:"devmode.sync"`
}

type syncRule struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

// $ docker inspect demo/buildpacks | jq -r '.[].Config.Labels["io.buildpacks.build.metadata"] | fromjson.bom[].metadata["devmode.sync"]'
func SyncRules(labels map[string]string) ([]*latest_v1.SyncRule, error) {
	metadataJSON, present := labels["io.buildpacks.build.metadata"]
	if !present {
		return nil, nil
	}

	m := buildMetadata{}
	if err := json.Unmarshal([]byte(metadataJSON), &m); err != nil {
		return nil, err
	}

	var rules []*latest_v1.SyncRule

	for _, b := range m.Bom {
		for _, sync := range b.Metadata.Sync {
			rules = append(rules, &latest_v1.SyncRule{
				Src:  sync.Src,
				Dest: sync.Dest,
			})
		}
	}

	return rules, nil
}
