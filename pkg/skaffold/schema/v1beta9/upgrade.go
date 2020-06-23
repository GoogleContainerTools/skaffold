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

package v1beta9

import (
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta10"
	pkgutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	incompatibleSyncWarning = `The semantics of sync has changed, the folder structure is no longer flattened but preserved (see https://skaffold.dev/docs/how-tos/filesync/). The likely impacted patterns in your skaffold yaml are: %s`
)

var (
	// compatibleSimplePattern have a directory prefix without stars and a basename with at most one star.
	compatibleSimplePattern = regexp.MustCompile(`^([^*]*/)?([^*/]*\*[^*/]*|[^*/]+)$`)
)

// Upgrade upgrades a configuration to the next version.
// Config changes from v1beta9 to v1beta10
// 1. Additions:
//    - DockerArtifact.NetworkMode
// 2. No removals
// 3. Updates:
//    - sync map becomes a list of sync rules
func (c *SkaffoldConfig) Upgrade() (util.VersionedConfig, error) {
	var newConfig next.SkaffoldConfig
	pkgutil.CloneThroughJSON(c, &newConfig)
	newConfig.APIVersion = next.Version

	err := util.UpgradePipelines(c, &newConfig, upgradeOnePipeline)
	return &newConfig, err
}

func upgradeOnePipeline(oldPipeline, newPipeline interface{}) error {
	oldBuild := &oldPipeline.(*Pipeline).Build
	newBuild := &newPipeline.(*next.Pipeline).Build

	// set Sync in newBuild
	newSyncRules := convertSyncRules(oldBuild.Artifacts)
	for i, a := range newBuild.Artifacts {
		if len(newSyncRules[i]) > 0 {
			a.Sync = &next.Sync{
				Manual: newSyncRules[i],
			}
		}
	}
	return nil
}

// convertSyncRules converts the old sync map into sync rules.
// It also prints a warning message when some rules can not be upgraded.
func convertSyncRules(artifacts []*Artifact) [][]*next.SyncRule {
	var incompatiblePatterns []string
	newSync := make([][]*next.SyncRule, len(artifacts))
	for i, a := range artifacts {
		newRules := make([]*next.SyncRule, 0, len(a.Sync))
		for src, dest := range a.Sync {
			var syncRule *next.SyncRule
			switch {
			case compatibleSimplePattern.MatchString(src):
				dest, strip := simplify(dest, compatibleSimplePattern.FindStringSubmatch(src)[1])
				syncRule = &next.SyncRule{
					Src:   src,
					Dest:  dest,
					Strip: strip,
				}
			case strings.Contains(src, "***"):
				dest, strip := simplify(dest, strings.Split(src, "***")[0])
				syncRule = &next.SyncRule{
					Src:   strings.Replace(src, "***", "**", -1),
					Dest:  dest,
					Strip: strip,
				}
			default:
				// Incompatible patterns contain `**` or glob directories.
				// Such patterns flatten the content at the destination which
				// cannot be reproduced with the current config. For example:
				// `/app/**/subdir/*.html`, `/app/*/*.html`
				incompatiblePatterns = append(incompatiblePatterns, src)
				syncRule = &next.SyncRule{
					Src:  src,
					Dest: dest,
				}
			}
			newRules = append(newRules, syncRule)
		}
		newSync[i] = newRules
		// blank input sync because it breaks cloning
		a.Sync = nil
	}
	if len(incompatiblePatterns) > 0 {
		logrus.Warnf(incompatibleSyncWarning, incompatiblePatterns)
	}
	return newSync
}

// simplify dest and strip, if strip is a suffix of dest modulo a trailing `/`.
func simplify(dest, strip string) (string, string) {
	if strip == "" || strip == "/" || dest == "" {
		return dest, strip
	}

	simpleStrip := strip
	simpleDest := dest

	if dest[len(dest)-1] != '/' {
		dest += "/"
	}

	if strings.HasSuffix(dest, strip) {
		simpleDest = strings.TrimSuffix(dest, strings.TrimPrefix(strip, "/"))
		simpleStrip = ""
		if simpleDest == "" {
			simpleDest = "."
		}
	}

	return simpleDest, simpleStrip
}
