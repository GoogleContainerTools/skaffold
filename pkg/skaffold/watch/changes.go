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

package watch

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type fileMap map[string]time.Time

// TODO(mrick): cached tree extension ala git
func stat(deps func() ([]string, error)) (fileMap, error) {
	state := fileMap{}
	paths, err := deps()
	if err != nil {
		return state, errors.Wrap(err, "listing files")
	}
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Debugf("could not stat dependency: %s", err)
				continue // Ignore files that don't exist
			}
			return nil, errors.Wrapf(err, "unable to stat file %s", path)
		}
		state[path] = stat.ModTime()
	}

	return state, nil
}

type Events struct {
	Added    []string
	Modified []string
	Deleted  []string
}

func (e Events) HasChanged() bool {
	return len(e.Added) != 0 || len(e.Deleted) != 0 || len(e.Modified) != 0
}

func (e *Events) String() string {
	added, deleted, modified := len(e.Added), len(e.Deleted), len(e.Modified)

	var sb strings.Builder
	if added > 0 {
		sb.WriteString(fmt.Sprintf("[watch event] added: %s\n", e.Added))
	}
	if deleted > 0 {
		sb.WriteString(fmt.Sprintf("[watch event] deleted: %s\n", e.Deleted))
	}
	if modified > 0 {
		sb.WriteString(fmt.Sprintf("[watch event] modified: %s\n", e.Modified))
	}
	return sb.String()
}

func events(prev, curr fileMap) Events {
	e := Events{}
	for f, t := range prev {
		modtime, ok := curr[f]
		if !ok {
			// file in prev but not in curr -> file deleted
			e.Deleted = append(e.Deleted, f)
			continue
		}
		if !modtime.Equal(t) {
			// file in both prev and curr
			// time not equal -> file modified
			e.Modified = append(e.Modified, f)
			continue
		}
	}

	for f := range curr {
		// don't need to check case where file is in both curr and prev
		// covered above
		_, ok := prev[f]
		if !ok {
			// file in curr but not in prev -> file added
			e.Added = append(e.Added, f)
		}
	}

	sortEvt(e)
	logrus.Debug(e.String())
	return e
}

func sortEvt(e Events) {
	sort.Strings(e.Added)
	sort.Strings(e.Modified)
	sort.Strings(e.Deleted)
}
