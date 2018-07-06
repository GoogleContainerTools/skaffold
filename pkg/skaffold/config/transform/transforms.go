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
				SetValueTemplates: release.SetValueTemplates,
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
