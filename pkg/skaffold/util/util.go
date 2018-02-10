/*
Copyright 2018 Google LLC

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

package util

import (
	"crypto/rand"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var Fs = afero.NewOsFs()

func ResetFs() {
	Fs = afero.NewOsFs()
}

func RandomID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func ExpandPaths(workspace string, paths []string) ([]string, error) {
	expandedPaths := map[string]struct{}{}
	for _, p := range paths {
		// If the path contains a filepath.Match wildcard, we have to walk the workspace
		// and find any matches to that pattern
		if containsWildcards(p) {
			logrus.Debugf("COPY or ADD directive with wildcard %s", p)
			if err := afero.Walk(Fs, workspace, func(fpath string, info os.FileInfo, err error) error {
				logrus.Debugf("expand: walk %s", fpath)
				if err != nil {
					return errors.Wrap(err, "getting relative path")
				}
				if match, _ := path.Match(p, fpath); !match {
					return nil
				}
				if err := addFileOrDir(Fs, fpath, info, expandedPaths); err != nil {
					return errors.Wrap(err, "adding file or directory")
				}
				return nil
			}); err != nil {
				return nil, errors.Wrap(err, "walking wildcard path")
			}
			continue
		}
		// If the path does not contain a wildcard, recursively add the directory or individual file
		info, err := Fs.Stat(p)
		if err != nil {
			return nil, errors.Wrap(err, "getting file info")
		}
		if err := addFileOrDir(Fs, p, info, expandedPaths); err != nil {
			return nil, errors.Wrap(err, "adding file or directory")
		}
	}
	ret := []string{}
	for ep := range expandedPaths {
		ret = append(ret, ep)
	}
	return ret, nil
}

func addFileOrDir(fs afero.Fs, ref string, info os.FileInfo, expandedPaths map[string]struct{}) error {
	if info.IsDir() {
		return addDir(fs, ref, expandedPaths)
	}
	expandedPaths[ref] = struct{}{}
	return nil
}

func addDir(fs afero.Fs, dir string, expandedPaths map[string]struct{}) error {
	logrus.Debugf("Recursively adding %s", dir)
	if err := afero.Walk(fs, dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		expandedPaths[path] = struct{}{}
		return nil
	}); err != nil {
		return errors.Wrap(err, "filepath walk")
	}
	return nil
}

func containsWildcards(path string) bool {
	for i := 0; i < len(path); i++ {
		ch := path[i]
		// These are the wildcards that correspond to filepath.Match
		if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}
