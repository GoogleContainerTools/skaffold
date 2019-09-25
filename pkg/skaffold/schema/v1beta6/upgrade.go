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

package v1beta6

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta7"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1beta6 to v1beta7
// 1. Additions:
// localdir/initImage
// helm useHelmSecrets
// 2. No removals
// 3. Updates:
// kaniko becomes cluster
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	// convert Deploy (should be the same)
	var newDeploy next.DeployConfig
	pkgutil.CloneThroughJSON(c.Deploy, &newDeploy)

	// convert Profiles (should be the same)
	var newProfiles []next.Profile
	if c.Profiles != nil {
		pkgutil.CloneThroughJSON(c.Profiles, &newProfiles)
	}

	// Update profile if kaniko build exists
	for i, p := range c.Profiles {
		upgradeKanikoBuild(p.Build, &newProfiles[i].Build)
	}

	// convert Kaniko if needed
	var newBuild next.BuildConfig
	pkgutil.CloneThroughJSON(c.Build, &newBuild)
	upgradeKanikoBuild(c.Build, &newBuild)

	// convert Test (should be the same)
	var newTest []*next.TestCase
	pkgutil.CloneThroughJSON(c.Test, &newTest)

	return &next.SkaffoldConfig{
		APIVersion: next.Version,
		Kind:       c.Kind,
		Build:      newBuild,
		Test:       newTest,
		Deploy:     newDeploy,
		Profiles:   newProfiles,
	}, nil
}

func upgradeKanikoBuild(build BuildConfig, newConfig *next.BuildConfig) {
	kaniko := build.KanikoBuild
	if kaniko == nil {
		return
	}

	// Else, transition values from old config to new config artifacts
	for _, a := range newConfig.Artifacts {
		pkgutil.CloneThroughJSON(kaniko, &a.KanikoArtifact)
	}
	// Transition values from old config to in cluster details
	pkgutil.CloneThroughJSON(kaniko, &newConfig.Cluster)
}
