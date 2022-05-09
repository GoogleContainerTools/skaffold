/*
Copyright 2022 The Skaffold Authors

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

package v2beta28

import (
	"encoding/json"

	"github.com/pkg/errors"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Upgrade upgrades a configuration to the next version.
// v2beta28 is the last config version for skaffold v1, and future version will
// follow the naming scheme of v3*
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig
	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline)
	return &newConfig, err
}

func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	oldPL := oldPipeline.(*Pipeline)
	newPL := newPipeline.(*next.Pipeline)

	// Copy kubectl deploy config to render config
	if oldPL.Deploy.KubectlDeploy != nil {
		newPL.Render.RawK8s = oldPL.Deploy.KubectlDeploy.Manifests
		newPL.Deploy.KubectlDeploy.Manifests = nil
	}

	// Copy kustomize deploy config to render config
	if oldPL.Deploy.KustomizeDeploy != nil {
		newPL.Render.Kustomize = oldPL.Deploy.KustomizeDeploy.KustomizePaths
		// nil out kustomize deployer as it shouldn't be a thing anymore
		newPL.Deploy.KustomizeDeploy = nil
	}

	// TODO(marlongamez): what should happen when migrating v2?
	// Copy Kpt deploy config to render config

	// Copy helm deploy config
	if oldPL.Deploy.HelmDeploy != nil {
		oldHelm, err := json.Marshal(*oldPL.Deploy.HelmDeploy)
		if err != nil {
			return errors.Wrap(err, "marshalling old helm deploy")
		}
		newHelm := next.LegacyHelmDeploy{}
		if err = json.Unmarshal(oldHelm, &newHelm); err != nil {
			return errors.Wrap(err, "unmarshalling into new helm deploy")
		}

		// Copy Releases and Flags into the render config
		newPL.Render.Helm = &next.Helm{}
		newPL.Render.Helm.Releases = newHelm.Releases
		newPL.Render.Helm.Flags = newHelm.Flags

		// Copy over lifecyle hooks for helm deployer
		newPL.Deploy.LegacyHelmDeploy = &next.LegacyHelmDeploy{}
		newPL.Deploy.LegacyHelmDeploy.LifecycleHooks = newHelm.LifecycleHooks
	}

	return nil
}
