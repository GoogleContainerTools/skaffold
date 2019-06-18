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

package v1alpha3

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1alpha3 to v1alpha4:
// 1. Additions:
//   - SkaffoldConfig.Test, Profile.Test, TestCase, TestConfig
//   - KanikoBuildContext.LocalDir, LocalDir
//   - KanikoBuild.Image
//   - Artifact.Sync
// 	 - JibMavenArtifact, JibGradleArtifact
// 2. No removal
// 3. Updates
//    - EnvTemplate.Template is now optional in yaml
//    - LocalBuild.SkipPush=false (v1alpha3) -> LocalBuild.Push=true (v1alpha4)_
//    - kustomizePath -> path in yaml
// 		- HelmRelease, HelmPackaged, HelmFQNConfig fields are optional in yaml,
//    - Artifact.imageName -> image, workspace -> context in yaml
//		- DockerArtifact.dockerfilePath -> dockerfile in yaml
//    - BazelArtifact.BuildTarget is optional in yaml
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	// convert Deploy (should be the same)
	var newDeploy next.DeployConfig
	if err := pkgutil.CloneThroughJSON(config.Deploy, &newDeploy); err != nil {
		return nil, errors.Wrap(err, "converting deploy config")
	}

	// convert Profiles (should be the same)
	var newProfiles []next.Profile
	if config.Profiles != nil {
		if err := pkgutil.CloneThroughJSON(config.Profiles, &newProfiles); err != nil {
			return nil, errors.Wrap(err, "converting new profile")
		}
		for i, oldProfile := range config.Profiles {
			convertBuild(oldProfile.Build, newProfiles[i].Build)
		}
	}

	// convert Build (should be the same)
	var newBuild next.BuildConfig
	oldBuild := config.Build
	if err := pkgutil.CloneThroughJSON(oldBuild, &newBuild); err != nil {
		return nil, errors.Wrap(err, "converting new build")
	}
	convertBuild(oldBuild, newBuild)

	return &next.SkaffoldConfig{
		APIVersion: next.Version,
		Kind:       config.Kind,
		Deploy:     newDeploy,
		Build:      newBuild,
		Profiles:   newProfiles,
	}, nil
}

func convertBuild(oldBuild BuildConfig, newBuild next.BuildConfig) {
	if oldBuild.LocalBuild != nil && oldBuild.LocalBuild.SkipPush != nil {
		push := !*oldBuild.LocalBuild.SkipPush
		newBuild.LocalBuild.Push = &push
	}
}
