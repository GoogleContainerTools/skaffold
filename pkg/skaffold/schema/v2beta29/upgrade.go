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

package v2beta29

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v3alpha1"
	pkgutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

var migrations = map[string]string{
	"/deploy/kubectl":             "/manifests/rawYaml",
	"/deploy/kustomize/paths":     "/manifests/kustomize/paths",
	"/deploy/kustomize/buildArgs": "/manifests/kustomize/buildArgs",
	"/deploy/helm":                "/manifests/helm",
}

// Upgrade upgrades a configuration to the next version.
// v2beta29 is the last config version for skaffold v1, and future version will
// follow the naming scheme of v3*
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig
	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline)
	if err != nil {
		return &newConfig, err
	}

	var newProfiles []next.Profile
	// seed with existing Profiles
	if c.Profiles != nil {
		pkgutil.CloneThroughJSON(newConfig.Profiles, &newProfiles)
	}

	// Update profiles patches
	for i, p := range c.Profiles {
		upgradePatches(p.Patches, newProfiles[i].Patches)
	}

	newConfig.Profiles = newProfiles

	return &newConfig, nil
}

func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	oldPL := oldPipeline.(*Pipeline)
	newPL := newPipeline.(*next.Pipeline)

	// Copy kubectl deploy config to render config
	if oldPL.Deploy.KubectlDeploy != nil {
		newPL.Render.RawK8s = oldPL.Deploy.KubectlDeploy.Manifests
	}

	// Copy kustomize deploy config to render config
	if oldPL.Deploy.KustomizeDeploy != nil {
		newPL.Render.Kustomize = &next.Kustomize{
			Paths:     oldPL.Deploy.KustomizeDeploy.KustomizePaths,
			BuildArgs: oldPL.Deploy.KustomizeDeploy.BuildArgs,
		}
		if len(newPL.Render.Kustomize.Paths) == 0 {
			newPL.Render.Kustomize.Paths = append(newPL.Render.Kustomize.Paths, ".")
		}
	}

	// Copy Kpt deploy config to render config
	if oldPL.Deploy.KptDeploy != nil {
		return errors.New("converting deploy.kpt isn't currently supported")
	}

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

		// Copy over removed artifactOverrides & imageStrategy field as identical setValues fields
		for i := 0; i < len(newPL.Render.Helm.Releases); i++ {
			// need to append, not override
			svs := map[string]string(oldPL.Deploy.HelmDeploy.Releases[i].SetValues)
			svts := map[string]string(oldPL.Deploy.HelmDeploy.Releases[i].SetValueTemplates)
			aos := map[string]string(oldPL.Deploy.HelmDeploy.Releases[i].ArtifactOverrides)
			if aos != nil && svs == nil {
				svs = map[string]string{}
			}
			if oldPL.Deploy.HelmDeploy.Releases[i].ImageStrategy.HelmConventionConfig != nil {
				// is 'helm' imageStrategy
				for k, v := range aos {
					// replace commonly used image name chars that are illegal helm template chars "/" & "-" with "_"
					validV := pkgutil.SanitizeHelmTemplateValue(v)
					if svts == nil {
						svts = map[string]string{}
					}
					svts[k+".tag"] = fmt.Sprintf("{{.IMAGE_TAG_%s}}", validV)
					svts[k+".repository"] = fmt.Sprintf("{{.IMAGE_REPO_%s}}", validV)
					if oldPL.Deploy.HelmDeploy.Releases[i].ImageStrategy.HelmConventionConfig.ExplicitRegistry {
						// is 'helm' imageStrategy + explicitRegistry
						svts[k+".registry"] = fmt.Sprintf("{{.IMAGE_DOMAIN_%s}}", validV)
						svts[k+".repository"] = fmt.Sprintf("{{.IMAGE_REPO_NO_DOMAIN_%s}}", validV)
					}
				}
			} else {
				// is 'fqn' imageStrategy
				for k, v := range aos {
					if svts == nil {
						svts = map[string]string{}
					}
					// replace commonly used image name chars that are illegal helm template chars "/" & "-" with "_"
					validV := pkgutil.SanitizeHelmTemplateValue(v)
					svts[k] = fmt.Sprintf("{{.IMAGE_FULLY_QUALIFIED_%s}}", validV)
				}
			}
			newPL.Render.Helm.Releases[i].SetValues = svs
			newPL.Render.Helm.Releases[i].SetValueTemplates = svts
		}

		// Copy over lifecyle hooks for helm deployer
		newPL.Deploy.LegacyHelmDeploy = &next.LegacyHelmDeploy{}
		newPL.Deploy.LegacyHelmDeploy.LifecycleHooks = newHelm.LifecycleHooks
	}
	return nil
}

func upgradePatches(olds []JSONPatch, news []next.JSONPatch) {
	for i, old := range olds {
		for str, repStr := range migrations {
			if strings.Contains(old.Path, str) {
				news[i].Path = strings.ReplaceAll(old.Path, str, repStr)
			}
			if strings.Contains(old.Path, "/deploy/kpt") {
				log.Entry(context.TODO()).Warn("skip migrating kpt deploy sections. Please migrate these over manually")
			}
		}
	}
}
