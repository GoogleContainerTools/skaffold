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
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var Fs = afero.NewOsFs()

func RandomID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

// These are the supported file formats for kubernetes manifests
var validSuffixes = []string{".yml", ".yaml", ".json"}

// IsSupportedKubernetesFormat is for determining if a file under a glob pattern
// is deployable file format. It makes no attempt to check whether or not the file
// is actually deployable or has the correct contents.
func IsSupportedKubernetesFormat(n string) bool {
	for _, s := range validSuffixes {
		if strings.HasSuffix(n, s) {
			return true
		}
	}
	return false
}

func StrSliceContains(sl []string, s string) bool {
	for _, a := range sl {
		if a == s {
			return true
		}
	}
	return false
}

func RelPathToAbsPath(relPaths []string) ([]string, error) {
	baseFs := afero.NewBasePathFs(Fs, "").(*afero.BasePathFs)

	absPath := []string{}
	for _, p := range relPaths {
		a := afero.FullBaseFsPath(baseFs, p)
		absPath = append(absPath, a)
	}
	return absPath, nil
}

// ExpandPaths uses a filepath.Match to expand paths according to wildcards.
// It requires a workspace directory, which is walked and tested for wildcard matches
// It is used by the dockerfile parser and you most likely want to use ExpandPathsGlob
func ExpandPaths(workspace string, paths []string) ([]string, error) {
	expandedPaths := map[string]struct{}{}
	for _, p := range paths {
		// If the path contains a filepath.Match wildcard, we have to walk the workspace
		// and find any matches to that pattern
		if containsWildcards(p) {
			logrus.Debugf("COPY or ADD directive with wildcard %s", p)
			if err := afero.Walk(Fs, workspace, func(fpath string, info os.FileInfo, err error) error {
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

// ExpandPathsGlob expands paths according to filepath.Glob patterns
// Returns a list of unique files that match the glob patterns passed in.
func ExpandPathsGlob(paths []string) ([]string, error) {
	expandedPaths := map[string]struct{}{}
	for _, p := range paths {
		if _, err := Fs.Stat(p); err == nil {
			// This is a file reference, so just add it
			expandedPaths[p] = struct{}{}
			continue
		}
		files, err := afero.Glob(Fs, p)
		if err != nil {
			return nil, errors.Wrap(err, "glob")
		}
		if files == nil {
			return nil, fmt.Errorf("File pattern must match at least one file %s", p)
		}

		for _, f := range files {
			fi, err := Fs.Stat(f)
			if err != nil {
				return nil, err
			}
			if err := addFileOrDir(Fs, f, fi, expandedPaths); err != nil {
				return nil, errors.Wrap(err, "adding file or dir")
			}
		}
	}
	ret := []string{}
	for k := range expandedPaths {
		ret = append(ret, k)
	}
	sort.Strings(ret)
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
		if info == nil {
			return nil
		}
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

// BoolPtr returns a pointer to a bool
func BoolPtr(b bool) *bool {
	o := b
	return &o
}

func ReadConfiguration(filename string) ([]byte, error) {
	switch {
	case filename == "":
		return nil, errors.New("filename not specified")
	case filename == "-":
		return ioutil.ReadAll(os.Stdin)
	case strings.HasPrefix(filename, "http://") || strings.HasPrefix(filename, "https://"):
		return download(filename)
	default:
		return ioutil.ReadFile(filename)
	}
}

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
