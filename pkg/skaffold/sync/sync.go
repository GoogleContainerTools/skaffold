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

package sync

import (
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
)

type Syncer interface {
	Sync(s *Item) error
}

type Item struct {
	Image  string
	Copy   map[string]string
	Delete map[string]string
}

func NewItem(a *latest.Artifact, e watch.Events) (*Item, error) {
	// If there are no changes, short circuit and don't sync anything
	if !e.HasChanged() || a.Sync == nil || len(a.Sync) == 0 {
		return nil, nil
	}

	toCopy, err := intersect(a.Workspace, a.Sync, append(e.Added, e.Modified...))
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and added, modified files")
	}

	toDelete, err := intersect(a.Workspace, a.Sync, e.Deleted)
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and added, modified files")
	}

	if toCopy == nil || toDelete == nil {
		return nil, nil
	}

	return &Item{
		Image:  a.ImageName,
		Copy:   toCopy,
		Delete: toDelete,
	}, nil
}

func intersect(context string, syncMap map[string]string, files []string) (map[string]string, error) {
	ret := map[string]string{}
	for _, f := range files {
		relPath, err := filepath.Rel(context, f)
		if err != nil {
			return nil, errors.Wrapf(err, "changed file %s is not relative to context %s", f, context)
		}
		for p, dst := range syncMap {
			match, err := filepath.Match(p, relPath)
			if err != nil {
				return nil, errors.Wrap(err, "pattern error")
			}
			if !match {
				return nil, nil
			}
			// If the source has special match characters,
			// the destination must be a directory
			if util.HasMeta(p) {
				dst = filepath.Join(dst, filepath.Base(relPath))
			}
			ret[f] = dst
		}
	}
	return ret, nil
}
