/*
Copyright 2020 The Skaffold Authors

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var syncLists = map[projectKey]SyncMap{}

type SyncMap map[string]SyncEntry

type SyncEntry struct {
	Dest     []string
	FileTime time.Time
	IsDirect bool
}

type JSONSyncMap struct {
	Direct    []JSONSyncEntry `json:"direct"`
	Generated []JSONSyncEntry `json:"generated"`
}

type JSONSyncEntry struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

var InitSync = initSync

func initSync(ctx context.Context, workspace string, a *latest.JibArtifact) error {
	syncMap, err := getSyncMapFunc(ctx, workspace, a)
	if err != nil {
		return fmt.Errorf("failed to initialize sync state for %q: %w", workspace, err)
	}
	syncLists[getProjectKey(workspace, a)] = *syncMap
	return nil
}

var GetSyncDiff = getSyncDiff

// returns toCopy, toDelete, error
func getSyncDiff(ctx context.Context, workspace string, a *latest.JibArtifact, e filemon.Events) (map[string][]string, map[string][]string, error) {
	// no deletions allowed
	if len(e.Deleted) != 0 {
		// change into logging
		logrus.Debug("Deletions are not supported by jib auto sync at the moment")
		return nil, nil, nil
	}

	// if anything that was modified was a build file, do NOT sync, do a rebuild
	buildFiles := GetBuildDefinitions(workspace, a)
	for _, f := range e.Modified {
		f, err := toAbs(f)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate absolute path: %w", err)
		}
		for _, bf := range buildFiles {
			if f == bf {
				return nil, nil, nil
			}
		}
	}

	currSyncMap := syncLists[getProjectKey(workspace, a)]

	// if we're only dealing with 1. modified and 2. directly syncable files,
	// then we can sync those files directly without triggering a build
	if len(e.Deleted) == 0 && len(e.Added) == 0 {
		matches := make(map[string][]string)
		for _, f := range e.Modified {
			f, err := toAbs(f)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to calculate absolute path: %w", err)
			}
			if val, ok := currSyncMap[f]; ok {
				if !val.IsDirect {
					break
				}
				matches[f] = val.Dest
				// if we decide that we don't need to do a call to getSyncMapFromSystem,
				// (which would update these file times), we have to update
				// our state for these files manually here.
				infog, err := os.Stat(f)
				if err != nil {
					return nil, nil, fmt.Errorf("could not obtain file mod time: %w", err)
				}
				val.FileTime = infog.ModTime()
				currSyncMap[f] = val
			} else {
				break
			}
		}
		if len(matches) == len(e.Modified) {
			return matches, nil, nil
		}
	}

	// we need to do another build and get a new sync map
	nextSyncMap, err := getSyncMapFunc(ctx, workspace, a)
	if err != nil {
		return nil, nil, err
	}
	syncLists[getProjectKey(workspace, a)] = *nextSyncMap

	toCopy := make(map[string][]string)

	// calculate the diff of the syncmaps
	for k, v := range *nextSyncMap {
		if curr, ok := currSyncMap[k]; ok {
			if v.FileTime != curr.FileTime {
				// file updated
				toCopy[k] = v.Dest
			}
		} else {
			// new file was created
			toCopy[k] = v.Dest
		}
	}

	return toCopy, nil, nil
}

// for testing
var (
	getSyncMapFunc = getSyncMap
)

func getSyncMap(ctx context.Context, workspace string, artifact *latest.JibArtifact) (*SyncMap, error) {
	// cmd will hold context that identifies the project
	cmd, err := getSyncMapCommand(ctx, workspace, artifact)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync command: %w", err)
	}

	sm, err := getSyncMapFromSystem(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain sync map from jib builder: %w", err)
	}
	return sm, nil
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
		return getSyncMapCommandGradle(ctx, workspace, artifact), nil
	default:
		return nil, fmt.Errorf("unable to handle Jib builder type %s for %s", t, workspace)
	}
}

func getSyncMapFromSystem(cmd *exec.Cmd) (*SyncMap, error) {
	jsm := JSONSyncMap{}
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jib sync map: %w", err)
	}

	matches := regexp.MustCompile(`BEGIN JIB JSON: SYNCMAP/1\r?\n({.*})`).FindSubmatch(stdout)
	if len(matches) == 0 {
		return nil, errors.New("failed to get Jib Sync data")
	}

	if err := json.Unmarshal(matches[1], &jsm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jib sync JSON: %w", err)
	}

	sm := make(SyncMap)
	if err := sm.addEntries(jsm.Direct, true); err != nil {
		return nil, fmt.Errorf("failed to add jib json direct entries to sync state: %w", err)
	}
	if err := sm.addEntries(jsm.Generated, false); err != nil {
		return nil, fmt.Errorf("failed to add jib json generated entries to sync state: %w", err)
	}
	return &sm, nil
}

func (sm SyncMap) addEntries(entries []JSONSyncEntry, direct bool) error {
	for _, entry := range entries {
		info, err := os.Stat(entry.Src)
		if err != nil {
			return fmt.Errorf("could not obtain file mod time for %q: %w", entry.Src, err)
		}
		sm[entry.Src] = SyncEntry{
			Dest:     []string{entry.Dest},
			FileTime: info.ModTime(),
			IsDirect: direct,
		}
	}
	return nil
}

func toAbs(f string) (string, error) {
	if !filepath.IsAbs(f) {
		af, err := filepath.Abs(f)
		if err != nil {
			return "", fmt.Errorf("failed to calculate absolute path: %w", err)
		}
		return af, nil
	}
	return f, nil
}
