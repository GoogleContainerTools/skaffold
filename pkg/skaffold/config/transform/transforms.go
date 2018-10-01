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
	"encoding/json"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ToV1Alpha2 transforms v1alpha1 configs to v1alpha2
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

// ToV1Alpha3 transforms configs from v1alpha2 to v1alpha3
func ToV1Alpha3(vc util.VersionedConfig) (util.VersionedConfig, error) {
	if vc.GetVersion() != v1alpha2.Version {
		return nil, fmt.Errorf("Incompatible version: %s", vc.GetVersion())
	}
	oldConfig := vc.(*v1alpha2.SkaffoldConfig)

	// convert v1alpha2.Deploy to v1alpha3.Deploy (should be the same)
	var newDeploy v1alpha3.DeployConfig
	if err := convert(oldConfig.Deploy, &newDeploy); err != nil {
		return nil, errors.Wrap(err, "converting deploy config")
	}
	// if the helm deploy config was set, then convert ValueFilePath to ValuesFiles
	if oldHelmDeploy := oldConfig.Deploy.DeployType.HelmDeploy; oldHelmDeploy != nil {
		for i, oldHelmRelease := range oldHelmDeploy.Releases {
			if oldHelmRelease.ValuesFilePath != "" {
				newDeploy.DeployType.HelmDeploy.Releases[i].ValuesFiles = []string{oldHelmRelease.ValuesFilePath}
			}
		}
	}

	// convert v1alpha2.Profiles to v1alpha3.Profiles (should be the same)
	var newProfiles []v1alpha3.Profile
	if oldConfig.Profiles != nil {
		if err := convert(oldConfig.Profiles, &newProfiles); err != nil {
			return nil, errors.Wrap(err, "converting new profile")
		}
	}
	// if the helm deploy config was set for a profile, then convert ValueFilePath to ValuesFiles
	for p, oldProfile := range oldConfig.Profiles {
		if oldProfileHelmDeploy := oldProfile.Deploy.DeployType.HelmDeploy; oldProfileHelmDeploy != nil {
			for i, oldProfileHelmRelease := range oldProfileHelmDeploy.Releases {
				if oldProfileHelmRelease.ValuesFilePath != "" {
					newProfiles[p].Deploy.DeployType.HelmDeploy.Releases[i].ValuesFiles = []string{oldProfileHelmRelease.ValuesFilePath}
				}
			}
		}
	}

	// convert v1alpha2.Build to v1alpha3.Build (different only for kaniko)
	oldKanikoBuilder := oldConfig.Build.KanikoBuild
	oldConfig.Build.KanikoBuild = nil

	// copy over old build config to new build config
	var newBuild v1alpha3.BuildConfig
	if err := convert(oldConfig.Build, &newBuild); err != nil {
		return nil, errors.Wrap(err, "converting new build")
	}
	// if the kaniko build was set, then convert it
	if oldKanikoBuilder != nil {
		newBuild.BuildType.KanikoBuild = &v1alpha3.KanikoBuild{
			BuildContext: v1alpha3.KanikoBuildContext{
				GCSBucket: oldKanikoBuilder.GCSBucket,
			},
			Namespace:      oldKanikoBuilder.Namespace,
			PullSecret:     oldKanikoBuilder.PullSecret,
			PullSecretName: oldKanikoBuilder.PullSecretName,
			Timeout:        oldKanikoBuilder.Timeout,
		}
	}
	newConfig := &v1alpha3.SkaffoldConfig{
		APIVersion: v1alpha3.Version,
		Kind:       oldConfig.Kind,
		Deploy:     newDeploy,
		Build:      newBuild,
		Profiles:   newProfiles,
	}
	return newConfig, nil
}

// ToV1Alpha4 transforms configs from v1alpha3 to v1alpha4
func ToV1Alpha4(vc util.VersionedConfig) (util.VersionedConfig, error) {
	if vc.GetVersion() != v1alpha3.Version {
		return nil, fmt.Errorf("Incompatible version: %s", vc.GetVersion())
	}
	oldConfig := vc.(*v1alpha3.SkaffoldConfig)

	// convert v1alpha3.Deploy to v1alpha4.Deploy (should be the same)
	var newDeploy v1alpha4.DeployConfig
	if err := convert(oldConfig.Deploy, &newDeploy); err != nil {
		return nil, errors.Wrap(err, "converting deploy config")
	}

	// convert v1alpha3.Profiles to v1alpha4.Profiles (should be the same)
	var newProfiles []v1alpha4.Profile
	if oldConfig.Profiles != nil {
		if err := convert(oldConfig.Profiles, &newProfiles); err != nil {
			return nil, errors.Wrap(err, "converting new profile")
		}
	}

	// convert v1alpha3.Build to v1alpha4.Build (should be the same)
	var newBuild v1alpha4.BuildConfig
	if err := convert(oldConfig.Build, &newBuild); err != nil {
		return nil, errors.Wrap(err, "converting new build")
	}

	return &v1alpha4.SkaffoldConfig{
		APIVersion: v1alpha4.Version,
		Kind:       oldConfig.Kind,
		Deploy:     newDeploy,
		Build:      newBuild,
		Profiles:   newProfiles,
	}, nil
}

func convert(old interface{}, new interface{}) error {
	o, err := json.Marshal(old)
	if err != nil {
		return errors.Wrap(err, "marshalling old")
	}
	if err := json.Unmarshal(o, &new); err != nil {
		return errors.Wrap(err, "unmarshalling new")
	}
	return nil
}
