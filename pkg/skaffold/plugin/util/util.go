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

package util

import "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"

// GroupArtifactsByEnvironment returns artifacts grouped by environment
func GroupArtifactsByEnvironment(artifacts []*latest.Artifact, env *latest.ExecutionEnvironment) map[*latest.ExecutionEnvironment][]*latest.Artifact {
	m := make(map[*latest.ExecutionEnvironment][]*latest.Artifact)
	for _, a := range artifacts {
		if a.ExecutionEnvironment == nil {
			m[env] = append(m[env], a)
			continue
		}
		m[a.ExecutionEnvironment] = append(m[a.ExecutionEnvironment], a)
	}
	return m
}
