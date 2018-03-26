/*
Copyright 2018 Google LLC

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

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/v1alpha2"
)

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

	newConfig := &v1alpha2.SkaffoldConfig{
		APIVersion: v1alpha2.Version,
		Kind:       oldConfig.Kind,
		Deploy:     oldConfig.Deploy,
		Build: v1alpha2.BuildConfig{
			Artifacts: oldConfig.Build.Artifacts,
			BuildType: oldConfig.Build.BuildType,
			TagPolicy: tagPolicy,
		},
	}
	return newConfig, nil
}
