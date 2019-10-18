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

package v1alpha1

import (
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1alpha1 to v1alpha2:
// 1. Additions
//  - Profiles
//	- BuildType.KanikoBuild
// 	- LocalBuild.useDockerCLI, useBuildkit
//  - GoogleCloudBuild.	DiskSizeGb, MachineType, Timeout, DockerImage
//  - DeployType.KustomizeDeploy
//  - KubectlDeploy.RemoteManifests, Flags - KubectlFlags type
//  - HelmRelease fields: setValues, setValueTemplates,wait,recreatePods,overrides,packaged,imageStrategy
//  - BazelArtifact introduced
//  - DockerArtifact fields: CacheFrom, Target
// 2. No removal
// 3. Updates
//  - TagPolicy is a struct
//
func (config *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var tagPolicy next.TagPolicy
	if config.Build.TagPolicy == "sha256" {
		tagPolicy = next.TagPolicy{
			ShaTagger: &next.ShaTagger{},
		}
	} else if config.Build.TagPolicy == "gitCommit" {
		tagPolicy = next.TagPolicy{
			GitTagger: &next.GitTagger{},
		}
	}

	var newHelmDeploy *next.HelmDeploy
	if config.Deploy.DeployType.HelmDeploy != nil {
		var newReleases []next.HelmRelease
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
		var newManifests []string
		logrus.Warn("Ignoring manifest parameters when transforming v1alpha1 config; check Kubernetes yaml before running skaffold")
		for _, manifest := range config.Deploy.DeployType.KubectlDeploy.Manifests {
			newManifests = append(newManifests, manifest.Paths...)
		}
		newKubectlDeploy = &next.KubectlDeploy{
			Manifests: newManifests,
		}
	}

	var newArtifacts []*next.Artifact
	for _, artifact := range config.Build.Artifacts {
		newArtifact := &next.Artifact{
			ImageName: artifact.ImageName,
			Workspace: artifact.Workspace,
		}

		if artifact.DockerfilePath != "" || len(artifact.BuildArgs) > 0 {
			newArtifact.ArtifactType = next.ArtifactType{
				DockerArtifact: &next.DockerArtifact{
					DockerfilePath: artifact.DockerfilePath,
					BuildArgs:      artifact.BuildArgs,
				},
			}
		}

		newArtifacts = append(newArtifacts, newArtifact)
	}

	newBuildType := next.BuildType{}
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
