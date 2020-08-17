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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
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

// orderedFileSet holds an ordered set of file paths.
type orderedFileSet struct {
	files []string
	seen  map[string]bool
}

func (l *orderedFileSet) Add(file string) {
	if l.seen[file] {
		return
	}

	if l.seen == nil {
		l.seen = make(map[string]bool)
	}
	l.seen[file] = true

	l.files = append(l.files, file)
}

func (l *orderedFileSet) Files() []string {
	return l.files
}

// ExpandPathsGlob expands paths according to filepath.Glob patterns
// Returns a list of unique files that match the glob patterns passed in.
func ExpandPathsGlob(workingDir string, paths []string) ([]string, error) {
	var set orderedFileSet

	for _, p := range paths {
		if filepath.IsAbs(p) {
			// This is a absolute file reference
			set.Add(p)
			continue
		}

		path := filepath.Join(workingDir, p)
		if _, err := os.Stat(path); err == nil {
			// This is a file reference, so just add it
			set.Add(path)
			continue
		}

		files, err := filepath.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("glob: %w", err)
		}
		if len(files) == 0 {
			logrus.Warnf("%s did not match any file", p)
		}

		for _, f := range files {
			if err := walk.From(f).WhenIsFile().Do(func(path string, _ walk.Dirent) error {
				set.Add(path)
				return nil
			}); err != nil {
				return nil, fmt.Errorf("filepath walk: %w", err)
			}
		}
	}

	return set.Files(), nil
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
			return fmt.Errorf("creating parent directory: %w", err)
		}
		if _, err = os.Create(path); err != nil {
			return fmt.Errorf("creating file: %w", err)
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
		return "", fmt.Errorf("%s is a directory", file)
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

// CloneThroughJSON clones an `old` object into a `new` one
// using json marshalling and unmarshalling.
// Since the object can be marshalled, it's almost sure it can be
// unmarshalled. So we prefer to panic instead of returning an error
// that would create an untestable branch on the call site.
func CloneThroughJSON(old interface{}, new interface{}) {
	o, err := json.Marshal(old)
	if err != nil {
		panic(fmt.Sprintf("marshalling old: %v", err))
	}
	if err := json.Unmarshal(o, new); err != nil {
		panic(fmt.Sprintf("unmarshalling new: %v", err))
	}
}

// CloneThroughYAML clones an `old` object into a `new` one
// using yaml marshalling and unmarshalling.
// Since the object can be marshalled, it's almost sure it can be
// unmarshalled. So we prefer to panic instead of returning an error
// that would create an untestable branch on the call site.
func CloneThroughYAML(old interface{}, new interface{}) {
	contents, err := yaml.Marshal(old)
	if err != nil {
		panic(fmt.Sprintf("marshalling old: %v", err))
	}
	if err := yaml.Unmarshal(contents, new); err != nil {
		panic(fmt.Sprintf("unmarshalling new: %v", err))
	}
}

// AbsolutePaths prepends each path in paths with workspace if the path isn't absolute
func AbsolutePaths(workspace string, paths []string) []string {
	var list []string

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspace, path)
		}
		list = append(list, path)
	}

	return list
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	// err could be permission-related
	return (err == nil || !os.IsNotExist(err)) && info.Mode().IsRegular()
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	// err could be permission-related
	return (err == nil || !os.IsNotExist(err)) && info.IsDir()
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

// Copies a file or directory tree.  There are 2x3 cases:
//   1. If _src_ is a file,
//      1. and _dst_ exists and is a file then _src_ is copied into _dst_
//      2. and _dst_ exists and is a directory, then _src_ is copied as _dst/$(basename src)_
//      3. and _dst_ does not exist, then _src_ is copied as _dst_.
//   2. If _src_ is a directory,
//      1. and _dst_ exists and is a file, then return an error
//      2. and _dst_ exists and is a directory, then src is copied as _dst/$(basename src)_
//      3. and _dst_ does not exist, then src is copied as _dst/src[1:]_.
func Copy(dst, src string) error {
	if IsFile(src) {
		switch {
		case IsFile(dst): // copy _src_ to _dst_
		case IsDir(dst): // copy _src_ to _dst/src[-1]
			dst = filepath.Join(dst, filepath.Base(src))
		default: // copy _src_ to _dst_
			if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
				return err
			}
		}
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		return err
	} else if !IsDir(src) {
		return errors.New("src does not exist")
	}
	// so src is a directory
	if IsFile(dst) {
		return errors.New("cannot copy directory into file")
	}
	srcPrefix := src
	if IsDir(dst) { // src is copied to _dst/$(basename src)
		srcPrefix = filepath.Dir(src)
	} else if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}
	return walk.From(src).Unsorted().WhenIsFile().Do(func(path string, _ walk.Dirent) error {
		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		destFile := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(destFile), os.ModePerm); err != nil {
			return err
		}

		out, err := os.Create(destFile)
		if err != nil {
			return err
		}

		_, err = io.Copy(out, in)
		return err
	})
}
