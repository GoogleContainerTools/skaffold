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

package v1beta8

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1beta8 to v1beta9
// 1. Additions:
//    gitTagger/variant
// 2. Removed all schemas associated with builder plugins
// 3. No updates
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	// convert Deploy (should be the same)
	var newDeploy latest.DeployConfig
	if err := pkgutil.CloneThroughJSON(config.Deploy, &newDeploy); err != nil {
		return nil, errors.Wrap(err, "converting deploy config")
	}

	// convert Profiles (should be the same)
	var newProfiles []latest.Profile
	if config.Profiles != nil {
		if err := pkgutil.CloneThroughJSON(config.Profiles, &newProfiles); err != nil {
			return nil, errors.Wrap(err, "converting new profile")
		}
	}

	// convert Build (should be same)
	var newBuild latest.BuildConfig
	if err := pkgutil.CloneThroughJSON(config.Build, &newBuild); err != nil {
		return nil, errors.Wrap(err, "converting new build")
	}

	for i, a := range config.Build.Artifacts {
		if a.BuilderPlugin == nil {
			continue
		}
		if a.BuilderPlugin.Name == "bazel" {
			var ba *latest.BazelArtifact
			contents, err := yaml.Marshal(a.BuilderPlugin.Properties)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling properties")
			}
			if err := yaml.Unmarshal(contents, &ba); err != nil {
				return nil, errors.Wrap(err, "unmarshalling bazel artifact")
			}
			newBuild.Artifacts[i].BazelArtifact = ba
		}

		if a.BuilderPlugin.Name == "docker" {
			var da *latest.DockerArtifact
			contents, err := yaml.Marshal(a.BuilderPlugin.Properties)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling properties")
			}
			if err := yaml.Unmarshal(contents, &da); err != nil {
				return nil, errors.Wrap(err, "unmarshalling bazel artifact")
			}
			newBuild.Artifacts[i].DockerArtifact = da
		}
	}

	// convert Test (should be the same)
	var newTest []*latest.TestCase
	if err := pkgutil.CloneThroughJSON(config.Test, &newTest); err != nil {
		return nil, errors.Wrap(err, "converting new test")
	}

	return &latest.SkaffoldConfig{
		APIVersion: latest.Version,
		Kind:       config.Kind,
		Pipeline: latest.Pipeline{
			Build:  newBuild,
			Test:   newTest,
			Deploy: newDeploy,
		},
		Profiles: newProfiles,
	}, nil
}
