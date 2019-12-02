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

package jib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

var syncLists = make(map[string]SyncMap)

type SyncMap struct {
	Direct    []SyncEntry `json:"direct"`
	Generated []SyncEntry `json:"generated"`
}

type SyncEntry struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`

	filetime time.Time
}

func InitSync(ctx context.Context, workspace string, artifact *latest.JibArtifact) error {
	syncMap, err := getSyncMap(ctx, workspace, artifact)
	if (err != nil) {
		return err
	}
	syncLists[artifact.Project] = *syncMap
	return nil
}

// returns toCopy, toDelete, error
func GetSyncDiff(ctx context.Context, workspace string, artifact *latest.JibArtifact, e filemon.Events) (map[string][]string, map[string][]string, error) {

	// if anything that was modified was a buildfile, do NOT sync, do a rebuild
	buildFiles := GetBuildDefinitions(artifact)
	for _, f := range e.Modified {
		if !filepath.IsAbs(f) {
			if ff, err := filepath.Abs(f); err != nil{
				return nil, nil, err
			} else {
				f = ff
			}
		}
		for _, bf := range buildFiles {
			if f == bf {
				return nil, nil, nil
			}
		}
	}

	// it seems less than efficient to keep the original JSON structure when doing look ups, so maybe we should only use the json objects for serialization
	// and store the sync data in a better type?
	oldSyncMap := syncLists[artifact.Project]


	// if all files are modified and direct, we don't need to build anything
	if len(e.Deleted) == 0 || len(e.Added) == 0 {
		matches := make(map[string][]string)
		for _, f := range e.Modified {
			for _, se := range oldSyncMap.Direct {
				// filemon.Events doesn't seem to make any guarantee about the paths,
				// so convert them to absolute paths (that's what jib provides)
				if !filepath.IsAbs(f) {
					if ff, err := filepath.Abs(f); err != nil{
						return nil, nil, err
					} else {
						f = ff
					}
				}
				if se.Src == f {
					matches[se.Src] = []string{se.Dest}
					break
				}
			}
		}
		if len(matches) == len(e.Modified) {
			return matches, nil, nil
		}
	}

	if len(e.Deleted) != 0 {
		// change into logging
		fmt.Println("Deletions are not supported by jib auto sync at the moment")
		return nil, nil, nil;
	}

	// we need to do another build and get a new sync map
	newSyncMap, err := getSyncMap(ctx, workspace, artifact)
	if err != nil {
		return nil, nil, err;
	}
	syncLists[artifact.Project] = *newSyncMap

	toCopy := make(map[string][]string)
	// calculate the diff of the syncmaps
	// known: this doesn't handle the case that something in the oldSyncMap is
	// no longer represented in the new sync map
	for _, se := range newSyncMap.Generated {
		for _, seOld := range oldSyncMap.Generated {
			if se.Src == seOld.Src && !se.filetime.Equal(seOld.filetime) {
				toCopy[se.Src] = []string{se.Dest}
			}
		}
	}
	for _, se := range newSyncMap.Direct {
		for _, seOld := range oldSyncMap.Direct {
			if se.Src == seOld.Src && !se.filetime.Equal(seOld.filetime) {
				toCopy[se.Src] = []string{se.Dest}
			}
		}
	}

	return toCopy, nil, nil
}

// getSyncMap returns a list of files that can be sync'd to a remote container
func getSyncMap(ctx context.Context, workspace string, artifact *latest.JibArtifact) (*SyncMap, error) {

	// cmd will hold context that identifies the project
	cmd, err := getSyncMapCommand(ctx, workspace, artifact)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	projectSyncMap := SyncMap{}
	if err = runAndParseSyncMap(cmd, &projectSyncMap); err != nil {
		return nil, errors.WithStack(err)
	}


	// store the filetimes for all these values
	if err := updateModTime(projectSyncMap.Direct); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := updateModTime(projectSyncMap.Generated); err != nil {
		return nil, errors.WithStack(err)
	}

	return &projectSyncMap, nil
}

func getSyncMapCommand(ctx context.Context, workspace string, artifact *latest.JibArtifact) (*exec.Cmd, error) {
	t, err := DeterminePluginType(workspace, artifact)
	if err != nil {
		return nil, err
	}

	switch t {
	case JibMaven:
		return getSyncMapCommandMaven(ctx, workspace, artifact), nil
	case JibGradle:
		return nil, errors.Errorf("unable to sync gradle projects at %s", workspace)
	default:
		return nil, errors.Errorf("unable to determine Jib builder type for %s", workspace)
	}
}

func runAndParseSyncMap(cmd *exec.Cmd, sm *SyncMap) error {
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to get Jib sync map")
	}

	// To parse the output, search for "BEGIN JIB JSON", then unmarshal the next line into the pathMap struct.
	matches := regexp.MustCompile(`BEGIN JIB JSON\r?\n({.*})`).FindSubmatch(stdout)
	if len(matches) == 0 {
		return errors.New("failed to get Jib Sync data")
	}

	line := bytes.Replace(matches[1], []byte(`\`), []byte(`\\`), -1)
	return json.Unmarshal(line, &sm)
}

func updateModTime(se []SyncEntry) error {
	for i, _ := range se {
		e := &se[i]
		if info, err := os.Stat(e.Src); err != nil {
			return errors.Wrap(err, "jib could not get filetime data")
		} else {
			e.filetime = info.ModTime();
		}
	}
	return nil
}

