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

package util

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
)

// ListBuilders returns a list of builder names being used in the given build config.
func ListBuilders(build *latest.BuildConfig) []string {
	if build == nil {
		return []string{}
	}

	results := util.NewStringSet()
	for _, artifact := range build.Artifacts {
		results.Insert(misc.ArtifactType(artifact))
	}

	return results.ToList()
}

// ListDeployers returns a list of deployer names being used in the given deploy config.
func ListDeployers(deploy *latest.DeployConfig) []string {
	if deploy == nil {
		return []string{}
	}

	return yamltags.GetYamlTags(deploy.DeployType)
}
