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

package util

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	hiddenPrefix string = "."
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
	return StrSliceIndex(sl, s) >= 0
}

func StrSliceIndex(sl []string, s string) int {
	for i, a := range sl {
		if a == s {
			return i
		}
	}
	return -1
}

func StrSliceInsert(sl []string, index int, insert []string) []string {
	newSlice := make([]string, len(sl)+len(insert))
	copy(newSlice[0:index], sl[0:index])
	copy(newSlice[index:index+len(insert)], insert)
	copy(newSlice[index+len(insert):], sl[index:])
	return newSlice
}

// ExpandPathsGlob expands paths according to filepath.Glob patterns
// Returns a list of unique files that match the glob patterns passed in.
func ExpandPathsGlob(workingDir string, paths []string) ([]string, error) {
	expandedPaths := make(map[string]bool)
	for _, p := range paths {
		if filepath.IsAbs(p) {
			// This is a absolute file reference
			expandedPaths[p] = true
			continue
		}

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
		if len(files) == 0 {
			logrus.Warnf("%s did not match any file", p)
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

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	o := s
	return &o
}

func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func Download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// VerifyOrCreateFile checks if a file exists at the given path,
// and if not, creates all parent directories and creates the file.
func VerifyOrCreateFile(path string) error {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		dir := filepath.Dir(path)
		if err = os.MkdirAll(dir, 0744); err != nil {
			return errors.Wrap(err, "creating parent directory")
		}
		if _, err = os.Create(path); err != nil {
			return errors.Wrap(err, "creating file")
		}
		return nil
	}
	return err
}

// RemoveFromSlice removes a string from a slice of strings
func RemoveFromSlice(s []string, target string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == target {
			s = append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// Expand replaces placeholders for a given key with a given value.
// It supports the ${key} and the $key syntax.
func Expand(text, key, value string) string {
	text = strings.Replace(text, "${"+key+"}", value, -1)

	indices := regexp.MustCompile(`\$`+key).FindAllStringIndex(text, -1)

	for i := len(indices) - 1; i >= 0; i-- {
		from := indices[i][0]
		to := indices[i][1]

		if to >= len(text) || !isAlphaNum(text[to]) {
			text = text[0:from] + value + text[to:]
		}
	}

	return text
}

func isAlphaNum(c uint8) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// AbsFile resolves the absolute path of the file named filename in directory workspace, erroring if it is not a file
func AbsFile(workspace string, filename string) (string, error) {
	file := filepath.Join(workspace, filename)
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.Errorf("%s is a directory", file)
	}
	return filepath.Abs(file)
}

// NonEmptyLines scans the provided input and returns the non-empty strings found as an array
func NonEmptyLines(input []byte) []string {
	var result []string
	scanner := bufio.NewScanner(bytes.NewReader(input))
	for scanner.Scan() {
		if line := scanner.Text(); len(line) > 0 {
			result = append(result, line)
		}
	}
	return result
}

// SHA256 returns the shasum of the contents of r
func SHA256(r io.Reader) (string, error) {
	hasher := sha256.New()
	_, err := io.Copy(hasher, r)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size()))), nil
}

// CloneThroughJSON marshals the old interface into the new one
func CloneThroughJSON(old interface{}, new interface{}) error {
	o, err := json.Marshal(old)
	if err != nil {
		return errors.Wrap(err, "marshalling old")
	}
	if err := json.Unmarshal(o, &new); err != nil {
		return errors.Wrap(err, "unmarshalling new")
	}
	return nil
}

// CloneThroughYAML marshals the old interface into the new one
func CloneThroughYAML(old interface{}, new interface{}) error {
	contents, err := yaml.Marshal(old)
	if err != nil {
		return errors.Wrap(err, "unmarshalling properties")
	}
	if err := yaml.Unmarshal(contents, new); err != nil {
		return errors.Wrap(err, "unmarshalling bazel artifact")
	}
	return nil
}

// AbsolutePaths prepends each path in paths with workspace if the path isn't absolute
func AbsolutePaths(workspace string, paths []string) []string {
	var p []string
	for _, path := range paths {
		// TODO(dgageot): this is only done for jib builder.
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspace, path)
		}
		p = append(p, path)
	}
	return p
}

// IsHiddenDir returns if a directory is hidden.
func IsHiddenDir(filename string) bool {
	// Return false for current dir
	if filename == hiddenPrefix {
		return false
	}
	return hasHiddenPrefix(filename)
}

// IsHiddenFile returns if a file is hidden.
// File is hidden if it starts with prefix "."
func IsHiddenFile(filename string) bool {
	return hasHiddenPrefix(filename)
}

func hasHiddenPrefix(s string) bool {
	return strings.HasPrefix(s, hiddenPrefix)
}
