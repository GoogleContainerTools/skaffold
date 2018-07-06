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

package transform

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/sirupsen/logrus"
)

func ToV1alpha3(vc util.VersionedConfig) (util.VersionedConfig, error) {
	if vc.GetVersion() != v1alpha1.Version {
		return nil, fmt.Errorf("Incompatible version: %s", vc.GetVersion())
	}
	oldConfig := vc.(*v1alpha2.SkaffoldConfig)

	var tagPolicy v1alpha3.TagPolicy
	if oldConfig.Build.TagPolicy == constants.TagStrategySha256 {
		tagPolicy = v1alpha3.TagPolicy{
			ShaTagger: &v1alpha3.ShaTagger{},
		}
	} else if oldConfig.Build.TagPolicy == constants.TagStrategyGitCommit {
		tagPolicy = v1alpha3.TagPolicy{
			GitTagger: &v1alpha3.GitTagger{},
		}
	}

	var newHelmDeploy *v1alpha3.HelmDeploy
	if oldConfig.Deploy.DeployType.HelmDeploy != nil {
		newReleases := make([]v1alpha3.HelmRelease, 0)
		for _, release := range oldConfig.Deploy.DeployType.HelmDeploy.Releases {
			newReleases = append(newReleases, v1alpha3.HelmRelease{
				Name:              release.Name,
				ChartPath:         release.ChartPath,
				Values:            release.Values,
				Namespace:         release.Namespace,
				Version:           release.Version,
				SetValues:         release.SetValues,
				SetValueTemplates: release.SetValuesTemplates,
				Wait:              release.Wait,
				Overrides:         release.Overrides,
				Packaged:          release.Packaged,
			})
		}
		newHelmDeploy = &v1alpha3.HelmDeploy{
			Releases: newReleases,
		}
	}
	return newConfig, nil
}
func ToV1Alpha2(vc util.VersionedConfig) (util.VersionedConfig, error) {
	if vc.GetVersion() != v1alpha1.Version {
		return nil, fmt.Errorf("Incompatible version: %s", vc.GetVersion())
	}
	oldConfig := vc.(*v1alpha1.SkaffoldConfig)

	var tagPolicy v1alpha2.TagPolicy
	if oldConfig.Build.TagPolicy == constants.TagStrategySha256 {
		tagPolicy = v1alpha2.TagPolicy{
			ShaTagger: &v1alpha2.ShaTagger{},
		}
	} else if oldConfig.Build.TagPolicy == constants.TagStrategyGitCommit {
		tagPolicy = v1alpha2.TagPolicy{
			GitTagger: &v1alpha2.GitTagger{},
		}
	}

	var newHelmDeploy *v1alpha2.HelmDeploy
	if oldConfig.Deploy.DeployType.HelmDeploy != nil {
		newReleases := make([]v1alpha2.HelmRelease, 0)
		for _, release := range oldConfig.Deploy.DeployType.HelmDeploy.Releases {
			newReleases = append(newReleases, v1alpha2.HelmRelease{
				Name:           release.Name,
				ChartPath:      release.ChartPath,
				ValuesFilePath: release.ValuesFilePath,
				Values:         release.Values,
				Namespace:      release.Namespace,
				Version:        release.Version,
			})
		}
		newHelmDeploy = &v1alpha2.HelmDeploy{
			Releases: newReleases,
		}
	}
	var newKubectlDeploy *v1alpha2.KubectlDeploy
	if oldConfig.Deploy.DeployType.KubectlDeploy != nil {
		newManifests := make([]string, 0)
		logrus.Warn("Ignoring manifest parameters when transforming v1alpha1 config; check kubernetes yaml before running skaffold")
		for _, manifest := range oldConfig.Deploy.DeployType.KubectlDeploy.Manifests {
			newManifests = append(newManifests, manifest.Paths...)
		}
		newKubectlDeploy = &v1alpha2.KubectlDeploy{
			Manifests: newManifests,
		}
	}

	var newArtifacts = make([]*v1alpha2.Artifact, 0)
	for _, artifact := range oldConfig.Build.Artifacts {
		newArtifacts = append(newArtifacts, &v1alpha2.Artifact{
			ImageName: artifact.ImageName,
			Workspace: artifact.Workspace,
			ArtifactType: v1alpha2.ArtifactType{
				DockerArtifact: &v1alpha2.DockerArtifact{
					DockerfilePath: artifact.DockerfilePath,
					BuildArgs:      artifact.BuildArgs,
				},
			},
		})
	}

	var newBuildType = v1alpha2.BuildType{}
	if oldConfig.Build.GoogleCloudBuild != nil {
		newBuildType.GoogleCloudBuild = &v1alpha2.GoogleCloudBuild{
			ProjectID: oldConfig.Build.GoogleCloudBuild.ProjectID,
		}
	}
	if oldConfig.Build.LocalBuild != nil {
		newBuildType.LocalBuild = &v1alpha2.LocalBuild{
			SkipPush: oldConfig.Build.LocalBuild.SkipPush,
		}
	}

	newConfig := &v1alpha2.SkaffoldConfig{
		APIVersion: v1alpha2.Version,
		Kind:       oldConfig.Kind,
		Deploy: v1alpha2.DeployConfig{
			DeployType: v1alpha2.DeployType{
				HelmDeploy:    newHelmDeploy,
				KubectlDeploy: newKubectlDeploy,
			},
		},
		Build: v1alpha2.BuildConfig{
			Artifacts: newArtifacts,
			BuildType: newBuildType,
			TagPolicy: tagPolicy,
		},
	}
	return newConfig, nil
}

func ToV1Alpha2(vc util.VersionedConfig) (util.VersionedConfig, error) {
	if vc.GetVersion() != v1alpha1.Version {
		return nil, fmt.Errorf("Incompatible version: %s", vc.GetVersion())
	}
	oldConfig := vc.(*v1alpha1.SkaffoldConfig)

	var tagPolicy v1alpha2.TagPolicy
	if oldConfig.Build.TagPolicy == constants.TagStrategySha256 {
		tagPolicy = v1alpha2.TagPolicy{
			ShaTagger: &v1alpha2.ShaTagger{},
		}
	} else if oldConfig.Build.TagPolicy == constants.TagStrategyGitCommit {
		tagPolicy = v1alpha2.TagPolicy{
			GitTagger: &v1alpha2.GitTagger{},
		}
	}

	var newHelmDeploy *v1alpha2.HelmDeploy
	if oldConfig.Deploy.DeployType.HelmDeploy != nil {
		newReleases := make([]v1alpha2.HelmRelease, 0)
		for _, release := range oldConfig.Deploy.DeployType.HelmDeploy.Releases {
			newReleases = append(newReleases, v1alpha2.HelmRelease{
				Name:           release.Name,
				ChartPath:      release.ChartPath,
				ValuesFilePath: release.ValuesFilePath,
				Values:         release.Values,
				Namespace:      release.Namespace,
				Version:        release.Version,
			})
		}
		newHelmDeploy = &v1alpha2.HelmDeploy{
			Releases: newReleases,
		}
	}
	var newKubectlDeploy *v1alpha2.KubectlDeploy
	if oldConfig.Deploy.DeployType.KubectlDeploy != nil {
		newManifests := make([]string, 0)
		logrus.Warn("Ignoring manifest parameters when transforming v1alpha1 config; check kubernetes yaml before running skaffold")
		for _, manifest := range oldConfig.Deploy.DeployType.KubectlDeploy.Manifests {
			newManifests = append(newManifests, manifest.Paths...)
		}
		newKubectlDeploy = &v1alpha2.KubectlDeploy{
			Manifests: newManifests,
		}
	}

	var newArtifacts = make([]*v1alpha2.Artifact, 0)
	for _, artifact := range oldConfig.Build.Artifacts {
		newArtifacts = append(newArtifacts, &v1alpha2.Artifact{
			ImageName: artifact.ImageName,
			Workspace: artifact.Workspace,
			ArtifactType: v1alpha2.ArtifactType{
				DockerArtifact: &v1alpha2.DockerArtifact{
					DockerfilePath: artifact.DockerfilePath,
					BuildArgs:      artifact.BuildArgs,
				},
			},
		})
	}

	var newBuildType = v1alpha2.BuildType{}
	if oldConfig.Build.GoogleCloudBuild != nil {
		newBuildType.GoogleCloudBuild = &v1alpha2.GoogleCloudBuild{
			ProjectID: oldConfig.Build.GoogleCloudBuild.ProjectID,
		}
	}
	if oldConfig.Build.LocalBuild != nil {
		newBuildType.LocalBuild = &v1alpha2.LocalBuild{
			SkipPush: oldConfig.Build.LocalBuild.SkipPush,
		}
	}

	newConfig := &v1alpha2.SkaffoldConfig{
		APIVersion: v1alpha2.Version,
		Kind:       oldConfig.Kind,
		Deploy: v1alpha2.DeployConfig{
			DeployType: v1alpha2.DeployType{
				HelmDeploy:    newHelmDeploy,
				KubectlDeploy: newKubectlDeploy,
			},
		},
		Build: v1alpha2.BuildConfig{
			Artifacts: newArtifacts,
			BuildType: newBuildType,
			TagPolicy: tagPolicy,
		},
	}
	return newConfig, nil
}
