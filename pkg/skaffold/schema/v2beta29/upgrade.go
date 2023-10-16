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
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v3alpha1"
	pkgutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

var artifactOverridesRegexp = regexp.MustCompile("/deploy/helm/releases/[0-9]+/artifactOverrides/image")

var migrations = map[string]string{
	"/deploy/kubectl/manifests":   "/manifests/rawYaml",
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

	var nProfiles []next.Profile
	// seed with existing Profiles
	if c.Profiles != nil {
		pkgutil.CloneThroughJSON(newConfig.Profiles, &nProfiles)
	}

	oldPatchesMap := map[string][]JSONPatch{}
	newPatchesMap := map[string][]next.JSONPatch{}
	for i, p := range c.Profiles {
		oldPatchesMap[p.Name], newPatchesMap[p.Name] = p.Patches, nProfiles[i].Patches
	}
	onePipelineUpgrader := NewOnePipelineUpgrader(oldPatchesMap, newPatchesMap)

	err := util.UpgradePipelines(c, &newConfig, onePipelineUpgrader.upgradeOnePipeline)
	if err != nil {
		return &newConfig, err
	}
	var newProfiles []next.Profile
	// seed with existing Profiles
	if c.Profiles != nil {
		pkgutil.CloneThroughJSON(newConfig.Profiles, &newProfiles)
	}

	for i := range newProfiles {
		if _, ok := newPatchesMap[newProfiles[i].Name]; ok {
			newProfiles[i].Patches = newPatchesMap[newProfiles[i].Name]
		}
	}

	// Update profiles patches
	for i, p := range c.Profiles {
		newProfiles[i].Patches = duplicateHelmPatches(newProfiles[i].Patches)
		upgradePatches(p.Patches, newProfiles[i].Patches)
	}

	newConfig.Profiles = newProfiles
	return &newConfig, nil
}

type OnePipelineUpgrader struct {
	oldPatchesMap map[string][]JSONPatch
	newPatchesMap map[string][]next.JSONPatch
}

func NewOnePipelineUpgrader(oldPatchesMap map[string][]JSONPatch, newPatchesMap map[string][]next.JSONPatch) OnePipelineUpgrader {
	return OnePipelineUpgrader{
		oldPatchesMap: oldPatchesMap,
		newPatchesMap: newPatchesMap,
	}
}

func (opu *OnePipelineUpgrader) upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
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

		// Inject Kubectl deployer to deploy kustomize manifests
		if err := mergeKustomizeIntoKubectlDeployer(newPL, oldPL); err != nil {
			return err
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

		// Copy Releases and Flags into the render config. Hooks are not copied due to v1
		// backwards compatibility: v1 skaffold render command doesn't trigger the hooks.
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
					svts[k+".tag"] = fmt.Sprintf("{{.IMAGE_TAG_%s}}@{{.IMAGE_DIGEST_%s}}", validV, validV)
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

		// Duplicate helm definitions between render and deployer.
		// This is required for backwards compatibility because skaffold v1 used helm namespace definitions for both
		// render (to populate {{.Release.Namespace}} templates) and deploy (as `--namespace` flag)
		newPL.Deploy.LegacyHelmDeploy = &next.LegacyHelmDeploy{}
		newPL.Deploy.LegacyHelmDeploy.LifecycleHooks = newHelm.LifecycleHooks
		newPL.Deploy.LegacyHelmDeploy.Releases = newHelm.Releases
		newPL.Deploy.LegacyHelmDeploy.Flags = newHelm.Flags
	}

	err := upgradeArtifactOverridesPatches(opu.oldPatchesMap, opu.newPatchesMap, oldPL)
	if err != nil {
		return fmt.Errorf("error converting helm artifactOverrides during version upgrade: %w", err)
	}

	return nil
}

func upgradeArtifactOverridesPatches(oldsMap map[string][]JSONPatch, newsMap map[string][]next.JSONPatch, oldPL *Pipeline) error {
	for k := range oldsMap {
		for i, old := range oldsMap[k] {
			if artifactOverridesRegexp.Match([]byte(old.Path)) {
				idx, err := strconv.Atoi(strings.Split(old.Path, "/")[4])
				if err != nil {
					return err
				}
				newPathModified := strings.ReplaceAll(strings.ReplaceAll(old.Path, "/deploy/helm", "/manifests/helm"),
					"artifactOverrides", "setValueTemplates")
				if oldPL.Deploy.HelmDeploy != nil {
					validV := pkgutil.SanitizeHelmTemplateValue(old.Value.Node.Value().(string))
					n := &util.YamlpatchNode{}

					if oldPL.Deploy.HelmDeploy.Releases[idx].ImageStrategy.HelmConventionConfig != nil {
						err = yaml.Unmarshal([]byte(fmt.Sprintf("\"{{.IMAGE_TAG_%s}}@{{.IMAGE_DIGEST_%s}}\"", validV, validV)), n)
						if err != nil {
							return err
						}

						newsMap[k][i] = next.JSONPatch{
							Op:    newsMap[k][i].Op,
							Path:  newPathModified + ".tag",
							From:  newsMap[k][i].From,
							Value: n,
						}

						if oldPL.Deploy.HelmDeploy.Releases[idx].ImageStrategy.HelmConventionConfig.ExplicitRegistry {
							// is 'helm' imageStrategy + explicitRegistry
							n := &util.YamlpatchNode{}
							err = yaml.Unmarshal([]byte(fmt.Sprintf("\"{{.IMAGE_DOMAIN_%s}}\"", validV)), n)
							if err != nil {
								return err
							}

							newsMap[k] = append(newsMap[k], next.JSONPatch{
								Op:    newsMap[k][i].Op,
								Path:  newPathModified + ".registry",
								From:  newsMap[k][i].From,
								Value: n,
							})

							n = &util.YamlpatchNode{}
							err := yaml.Unmarshal([]byte(fmt.Sprintf("\"{{.IMAGE_REPO_NO_DOMAIN_%s}}\"", validV)), n)
							if err != nil {
								return err
							}

							newsMap[k] = append(newsMap[k], next.JSONPatch{
								Op:    newsMap[k][i].Op,
								Path:  newPathModified + ".repository",
								From:  newsMap[k][i].From,
								Value: n,
							})
						} else {
							// is 'helm' imageStrategy
							n = &util.YamlpatchNode{}
							err := yaml.Unmarshal([]byte(fmt.Sprintf("\"{{.IMAGE_REPO_%s}}\"", validV)), n)
							if err != nil {
								return err
							}

							newsMap[k] = append(newsMap[k], next.JSONPatch{

								Op:    newsMap[k][i].Op,
								Path:  newPathModified + ".repository",
								From:  newsMap[k][i].From,
								Value: n,
							})
						}
					} else {
						// is 'fqn' imageStrategy
						// replace commonly used image name chars that are illegal helm template chars "/" & "-" with "_"
						newValue := fmt.Sprintf("\"{{.IMAGE_FULLY_QUALIFIED_%s}}\"", validV)

						n := &util.YamlpatchNode{}
						err := yaml.Unmarshal([]byte(newValue), n)
						if err != nil {
							return err
						}
						newsMap[k][i].Value = n
						newsMap[k][i].Path = newPathModified
					}
				}
			}
		}
	}
	return nil
}

func upgradePatches(olds []JSONPatch, news []next.JSONPatch) {
	for i, old := range olds {
		for str, repStr := range migrations {
			// For the following cases:
			// 1. this is handled by upgradeArtifactOverridesPatches.
			// 2. We don't update Helm deployer hook patches due to they shouldn only be present in the deploy stanza, not in the manifest.
			if artifactOverridesRegexp.Match([]byte(old.Path)) || isHelmDeployerHookPatch(old.Path) {
				continue
			}
			if strings.Contains(old.Path, str) {
				news[i].Path = strings.ReplaceAll(old.Path, str, repStr)
			}
			if strings.Contains(old.Path, "/deploy/kpt") {
				log.Entry(context.TODO()).Warn("skip migrating kpt deploy sections. Please migrate these over manually")
			}
		}
	}
}

func mergeKustomizeIntoKubectlDeployer(newPL *next.Pipeline, oldPL *Pipeline) error {
	kustomizeD := &next.KubectlDeploy{}
	pkgutil.CloneThroughJSON(oldPL.Deploy.KustomizeDeploy, kustomizeD)

	if newPL.Deploy.KubectlDeploy == nil {
		newPL.Deploy.KubectlDeploy = kustomizeD
	} else {
		kubectlD := newPL.Deploy.KubectlDeploy

		if kubectlD.DefaultNamespace != nil && kustomizeD.DefaultNamespace != nil && *(kubectlD.DefaultNamespace) != *(kustomizeD.DefaultNamespace) {
			return errors.New("can't merge defaultNamespace property from kustomize into kubectl deployer, property is already set with different value")
		}

		if kubectlD.Flags.DisableValidation != kustomizeD.Flags.DisableValidation {
			return errors.New("can't merge disableValidation property from kustomize into kubectl deployer, property is already set with different value")
		}

		if kustomizeD.DefaultNamespace != nil {
			kubectlD.DefaultNamespace = kustomizeD.DefaultNamespace
		}

		kubectlD.Flags = next.KubectlFlags{
			DisableValidation: kustomizeD.Flags.DisableValidation,
			Global:            append(kubectlD.Flags.Global, kustomizeD.Flags.Global...),
			Apply:             append(kubectlD.Flags.Apply, kustomizeD.Flags.Apply...),
			Delete:            append(kubectlD.Flags.Delete, kustomizeD.Flags.Delete...),
		}

		kubectlD.LifecycleHooks = next.DeployHooks{
			PreHooks:  append(kubectlD.LifecycleHooks.PreHooks, kustomizeD.LifecycleHooks.PreHooks...),
			PostHooks: append(kubectlD.LifecycleHooks.PostHooks, kustomizeD.LifecycleHooks.PostHooks...),
		}
	}

	return nil
}

// duplicate the original helm profile patches to the end
func duplicateHelmPatches(patches []next.JSONPatch) []next.JSONPatch {
	var duplicates []next.JSONPatch
	for i := range patches {
		if !strings.Contains(patches[i].Path, "/deploy/helm") || isHelmDeployerHookPatch(patches[i].Path) {
			continue
		}
		duplicates = append(duplicates, patches[i])
	}
	return append(patches, duplicates...)
}

func isHelmDeployerHookPatch(patchPath string) bool {
	return strings.Contains(patchPath, "/deploy/helm/hooks")
}
