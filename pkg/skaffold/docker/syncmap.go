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

package docker

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
)

// SyncMap creates a map of syncable files by looking at the COPY/ADD commands in the Dockerfile.
// All keys are relative to the Skaffold root, the destinations are absolute container paths.
// TODO(corneliusweig) destinations are not resolved across stages in multistage dockerfiles. Is there a use-case for that?
func SyncMap(ctx context.Context, workspace string, dockerfilePath string, buildArgs map[string]*string, insecureRegistries map[string]bool) (map[string][]string, error) {
	absDockerfilePath, err := NormalizeDockerfilePath(workspace, dockerfilePath)
	if err != nil {
		return nil, errors.Wrap(err, "normalizing dockerfile path")
	}

	// only the COPY/ADD commands from the last image are syncable
	fts, err := readCopyCmdsFromDockerfile(true, absDockerfilePath, workspace, buildArgs, insecureRegistries)
	if err != nil {
		return nil, err
	}

	excludes, err := readDockerignore(workspace)
	if err != nil {
		return nil, errors.Wrap(err, "reading .dockerignore")
	}

	srcByDest, err := walkWorkspaceWithDestinations(workspace, excludes, fts)
	if err != nil {
		return nil, errors.Wrap(err, "walking workspace")
	}

	return invertMap(srcByDest), nil
}

// walkWorkspaceWithDestinations walks the given host directories and determines their
// location in the container. It returns a map of host path by container destination.
// Note: if you change this function, you might also want to modify `WalkWorkspace`.
func walkWorkspaceWithDestinations(workspace string, excludes []string, fts []fromTo) (map[string]string, error) {
	pExclude, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return nil, errors.Wrap(err, "invalid exclude patterns")
	}

	// Walk the workspace
	srcByDest := make(map[string]string)
	for _, ft := range fts {
		absFrom := filepath.Join(workspace, ft.from)

		fi, err := os.Stat(absFrom)
		if err != nil {
			return nil, errors.Wrapf(err, "stating file %s", absFrom)
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			if err := godirwalk.Walk(absFrom, &godirwalk.Options{
				Unsorted: true,
				Callback: func(fpath string, info *godirwalk.Dirent) error {
					if fpath == absFrom {
						return nil
					}

					relPath, err := filepath.Rel(workspace, fpath)
					if err != nil {
						return err
					}

					ignored, err := pExclude.Matches(relPath)
					if err != nil {
						return err
					}

					if info.IsDir() {
						if ignored {
							return filepath.SkipDir
						}
					} else if !ignored {
						relBase, err := filepath.Rel(absFrom, fpath)
						if err != nil {
							return err
						}
						srcByDest[path.Join(ft.to, filepath.ToSlash(relBase))] = relPath
					}

					return nil
				},
			}); err != nil {
				return nil, errors.Wrapf(err, "walking folder %s", absFrom)
			}
		case mode.IsRegular():
			ignored, err := pExclude.Matches(ft.from)
			if err != nil {
				return nil, err
			}

			if !ignored {
				base := filepath.Base(ft.from)
				if ft.toIsDir {
					srcByDest[path.Join(ft.to, base)] = ft.from
				} else {
					srcByDest[ft.to] = ft.from
				}
			}
		}
	}

	return srcByDest, nil
}

func invertMap(kv map[string]string) map[string][]string {
	// len(kv) is a good upper bound for the size, because most files will have exactly one destination
	keysByValue := make(map[string][]string, len(kv))
	for k, v := range kv {
		if vs, ok := keysByValue[v]; ok {
			keysByValue[v] = append(vs, k)
		} else {
			keysByValue[v] = []string{k}
		}
	}
	return keysByValue
}
