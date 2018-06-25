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

package util

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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

func UniqueStrSlice(values []string) []string {
	var unique []string

	m := make(map[string]bool)
	for _, value := range values {
		m[value] = true
	}
	for value := range m {
		unique = append(unique, value)
	}

	sort.Strings(unique)
	return unique
}

// ExpandPathsGlob expands paths according to filepath.Glob patterns
// Returns a list of unique files that match the glob patterns passed in.
func ExpandPathsGlob(workingDir string, paths []string) ([]string, error) {
	expandedPaths := make(map[string]bool)
	for _, p := range paths {
		path := filepath.Join(workingDir, p)

		if _, err := os.Stat(path); err == nil {
			// This is a file reference, so just add it
			expandedPaths[path] = true
			continue
		}

		files, err := filepath.Glob(path)
		if err != nil {
			return nil, errors.Wrap(err, "glob")
		}
		if files == nil {
			return nil, fmt.Errorf("File pattern must match at least one file %s", path)
		}

		for _, f := range files {
			err := filepath.Walk(f, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					expandedPaths[path] = true
				}

				return nil
			})
			if err != nil {
				return nil, errors.Wrap(err, "filepath walk")
			}
		}
	}

	var ret []string
	for k := range expandedPaths {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret, nil
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
		directory := filepath.Dir(filename)
		baseName := filepath.Base(filename)
		if baseName != "skaffold.yaml" {
			return ioutil.ReadFile(filename)
		}
		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			logrus.Infof("Could not open skaffold.yaml: \"%s\"", err)
			logrus.Infof("Trying to read from skaffold.yml instead")
			return ioutil.ReadFile(filepath.Join(directory, "skaffold.yml"))
		}
		return contents, err
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
