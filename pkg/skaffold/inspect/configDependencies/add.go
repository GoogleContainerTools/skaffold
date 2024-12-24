/*
Copyright 2023 The Skaffold Authors

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

package inspect

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type configDependencyList struct {
	Dependencies []configDependencyEntry
}

type configDependencyEntry struct {
	Names                  []string                `json:"configs,omitempty"`
	Path                   string                  `json:"path,omitempty"`
	Git                    *git                    `json:"git,omitempty"`
	GoogleCloudStorage     *googleCloudStorage     `json:"googleCloudStorage,omitempty"`
	GoogleCloudBuildRepoV2 *googleCloudBuildRepoV2 `json:"googleCloudBuildRepoV2,omitempty"`
	ActiveProfiles         []activeProfile         `json:"activeProfiles,omitempty"`
}

type git struct {
	Repo string `json:"repo"`
	Path string `json:"path,omitempty"`
	Ref  string `json:"ref,omitempty"`
	Sync bool   `json:"sync,omitempty"`
}

type googleCloudStorage struct {
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Sync   bool   `json:"sync,omitempty"`
}

type googleCloudBuildRepoV2 struct {
	ProjectID  string `json:"projectID"`
	Region     string `json:"region"`
	Connection string `json:"connection"`
	Repo       string `json:"repo"`
	Path       string `json:"path,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Sync       bool   `json:"sync,omitempty"`
}

type activeProfile struct {
	Name        string   `json:"name"`
	ActivatedBy []string `json:"activatedBy,omitempty"`
}

func AddConfigDependencies(ctx context.Context, out io.Writer, opts inspect.Options, inputFile string) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)

	jsonFile, err := os.Open(inputFile)
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	defer jsonFile.Close()
	fileBytes, err := io.ReadAll(jsonFile)
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	var cds configDependencyList
	if err := json.Unmarshal(fileBytes, &cds); err != nil {
		formatter.WriteErr(err)
		return err
	}
	cfgDependencies := convertToLatestConfigDependencies(cds)

	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		RemoteCacheDir:      opts.RemoteCacheDir,
		ConfigurationFilter: opts.Modules,
		SkipConfigDefaults:  true,
		MakePathsAbsolute:   util.Ptr(false),
	})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}

	for _, cfg := range cfgs {
		cfg.Dependencies = append(cfg.Dependencies, cfgDependencies...)
	}
	return inspect.MarshalConfigSet(cfgs)
}

func convertToLatestConfigDependencies(cfgDepList configDependencyList) []latest.ConfigDependency {
	var res []latest.ConfigDependency
	for _, d := range cfgDepList.Dependencies {
		var cd latest.ConfigDependency
		cd.Names = d.Names
		cd.Path = d.Path
		if d.Git != nil {
			cd.GitRepo = &latest.GitInfo{
				Repo: d.Git.Repo,
				Path: d.Git.Path,
				Ref:  d.Git.Ref,
				Sync: &d.Git.Sync,
			}
		}
		if d.GoogleCloudStorage != nil {
			cd.GoogleCloudStorage = &latest.GoogleCloudStorageInfo{
				Source: d.GoogleCloudStorage.Source,
				Path:   d.GoogleCloudStorage.Path,
				Sync:   &d.GoogleCloudStorage.Sync,
			}
		}
		if d.GoogleCloudBuildRepoV2 != nil {
			cd.GoogleCloudBuildRepoV2 = &latest.GoogleCloudBuildRepoV2Info{
				ProjectID:  d.GoogleCloudBuildRepoV2.ProjectID,
				Region:     d.GoogleCloudBuildRepoV2.Region,
				Connection: d.GoogleCloudBuildRepoV2.Connection,
				Repo:       d.GoogleCloudBuildRepoV2.Repo,
				Path:       d.GoogleCloudBuildRepoV2.Path,
				Ref:        d.GoogleCloudBuildRepoV2.Ref,
				Sync:       &d.GoogleCloudBuildRepoV2.Sync,
			}
		}
		var profileDep []latest.ProfileDependency
		for _, ap := range d.ActiveProfiles {
			profileDep = append(profileDep, latest.ProfileDependency{
				Name:        ap.Name,
				ActivatedBy: ap.ActivatedBy,
			})
		}
		cd.ActiveProfiles = profileDep

		res = append(res, cd)
	}
	return res
}
