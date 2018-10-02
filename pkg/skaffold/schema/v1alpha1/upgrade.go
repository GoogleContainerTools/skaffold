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

package v1alpha1

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/sirupsen/logrus"
)

// Upgrade upgrades a configuration to the next version.
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var tagPolicy next.TagPolicy
	if config.Build.TagPolicy == constants.TagStrategySha256 {
		tagPolicy = next.TagPolicy{
			ShaTagger: &next.ShaTagger{},
		}
	} else if config.Build.TagPolicy == constants.TagStrategyGitCommit {
		tagPolicy = next.TagPolicy{
			GitTagger: &next.GitTagger{},
		}
	}

	var newHelmDeploy *next.HelmDeploy
	if config.Deploy.DeployType.HelmDeploy != nil {
		newReleases := make([]next.HelmRelease, 0)
		for _, release := range config.Deploy.DeployType.HelmDeploy.Releases {
			newReleases = append(newReleases, next.HelmRelease{
				Name:           release.Name,
				ChartPath:      release.ChartPath,
				ValuesFilePath: release.ValuesFilePath,
				Values:         release.Values,
				Namespace:      release.Namespace,
				Version:        release.Version,
			})
		}
		newHelmDeploy = &next.HelmDeploy{
			Releases: newReleases,
		}
	}
	var newKubectlDeploy *next.KubectlDeploy
	if config.Deploy.DeployType.KubectlDeploy != nil {
		newManifests := make([]string, 0)
		logrus.Warn("Ignoring manifest parameters when transforming v1alpha1 config; check kubernetes yaml before running skaffold")
		for _, manifest := range config.Deploy.DeployType.KubectlDeploy.Manifests {
			newManifests = append(newManifests, manifest.Paths...)
		}
		newKubectlDeploy = &next.KubectlDeploy{
			Manifests: newManifests,
		}
	}

	var newArtifacts = make([]*next.Artifact, 0)
	for _, artifact := range config.Build.Artifacts {
		newArtifacts = append(newArtifacts, &next.Artifact{
			ImageName: artifact.ImageName,
			Workspace: artifact.Workspace,
			ArtifactType: next.ArtifactType{
				DockerArtifact: &next.DockerArtifact{
					DockerfilePath: artifact.DockerfilePath,
					BuildArgs:      artifact.BuildArgs,
				},
			},
		})
	}

	var newBuildType = next.BuildType{}
	if config.Build.GoogleCloudBuild != nil {
		newBuildType.GoogleCloudBuild = &next.GoogleCloudBuild{
			ProjectID: config.Build.GoogleCloudBuild.ProjectID,
		}
	}
	if config.Build.LocalBuild != nil {
		newBuildType.LocalBuild = &next.LocalBuild{
			SkipPush: config.Build.LocalBuild.SkipPush,
		}
	}

	return &next.SkaffoldConfig{
		APIVersion: next.Version,
		Kind:       config.Kind,
		Deploy: next.DeployConfig{
			DeployType: next.DeployType{
				HelmDeploy:    newHelmDeploy,
				KubectlDeploy: newKubectlDeploy,
			},
		},
		Build: next.BuildConfig{
			Artifacts: newArtifacts,
			BuildType: newBuildType,
			TagPolicy: tagPolicy,
		},
	}, nil
}
