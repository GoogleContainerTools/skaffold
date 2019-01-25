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
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ResolveEnvironmentReplacementList resolves a list of values by calling resolveEnvironmentReplacement
func ResolveEnvironmentReplacementList(values, envs []string, isFilepath bool) ([]string, error) {
	var resolvedValues []string
	for _, value := range values {
		if IsSrcRemoteFileURL(value) {
			resolvedValues = append(resolvedValues, value)
			continue
		}
		resolved, err := ResolveEnvironmentReplacement(value, envs, isFilepath)
		logrus.Debugf("Resolved %s to %s", value, resolved)
		if err != nil {
			return nil, err
		}
		resolvedValues = append(resolvedValues, resolved)
	}
	return resolvedValues, nil
}

// ResolveEnvironmentReplacement resolves replacing env variables in some text from envs
// It takes in a string representation of the command, the value to be resolved, and a list of envs (config.Env)
// Ex: fp = $foo/newdir, envs = [foo=/foodir], then this should return /foodir/newdir
// The dockerfile/shell package handles processing env values
// It handles escape characters and supports expansion from the config.Env array
// Shlex handles some of the following use cases (these and more are tested in integration tests)
// ""a'b'c"" -> "a'b'c"
// "Rex\ The\ Dog \" -> "Rex The Dog"
// "a\"b" -> "a"b"
func ResolveEnvironmentReplacement(value string, envs []string, isFilepath bool) (string, error) {
	shlex := shell.NewLex(parser.DefaultEscapeToken)
	fp, err := shlex.ProcessWord(value, envs)
	if !isFilepath {
		return fp, err
	}
	if err != nil {
		return "", err
	}
	fp = filepath.Clean(fp)
	if IsDestDir(value) && !IsDestDir(fp) {
		fp = fp + "/"
	}
	return fp, nil
}

// ContainsWildcards returns true if any entry in paths contains wildcards
func ContainsWildcards(paths []string) bool {
	for _, path := range paths {
		if strings.ContainsAny(path, "*?[") {
			return true
		}
	}
	return false
}

// ResolveSources resolves the given sources if the sources contains wildcards
// It returns a list of resolved sources
func ResolveSources(srcsAndDest instructions.SourcesAndDest, root string) ([]string, error) {
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	// If sources contain wildcards, we first need to resolve them to actual paths
	if ContainsWildcards(srcs) {
		logrus.Debugf("Resolving srcs %v...", srcs)
		files, err := RelativeFiles("", root)
		if err != nil {
			return nil, err
		}
		srcs, err = matchSources(srcs, files)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Resolved sources to %v", srcs)
	}
	// Check to make sure the sources are valid
	return srcs, IsSrcsValid(srcsAndDest, srcs, root)
}

// matchSources returns a list of sources that match wildcards
func matchSources(srcs, files []string) ([]string, error) {
	var matchedSources []string
	for _, src := range srcs {
		if IsSrcRemoteFileURL(src) {
			matchedSources = append(matchedSources, src)
			continue
		}
		src = filepath.Clean(src)
		for _, file := range files {
			if filepath.IsAbs(src) {
				file = filepath.Join(constants.RootDir, file)
			}
			matched, err := filepath.Match(src, file)
			if err != nil {
				return nil, err
			}
			if matched || src == file {
				matchedSources = append(matchedSources, file)
			}
		}
	}
	return matchedSources, nil
}

func IsDestDir(path string) bool {
	// try to stat the path
	fileInfo, err := os.Stat(path)
	if err != nil {
		// fall back to string-based determination
		return strings.HasSuffix(path, "/") || path == "."
	}
	// if it's a real path, check the fs response
	return fileInfo.IsDir()
}

// DestinationFilepath returns the destination filepath from the build context to the image filesystem
// If source is a file:
//	If dest is a dir, copy it to /dest/relpath
// 	If dest is a file, copy directly to dest
// If source is a dir:
//	Assume dest is also a dir, and copy to dest/relpath
// If dest is not an absolute filepath, add /cwd to the beginning
func DestinationFilepath(src, dest, cwd string) (string, error) {
	if IsDestDir(dest) {
		destPath := filepath.Join(dest, filepath.Base(src))
		if filepath.IsAbs(dest) {
			return destPath, nil
		}
		return filepath.Join(cwd, destPath), nil
	}
	if filepath.IsAbs(dest) {
		return dest, nil
	}
	return filepath.Join(cwd, dest), nil
}

// URLDestinationFilepath gives the destination a file from a remote URL should be saved to
func URLDestinationFilepath(rawurl, dest, cwd string) string {
	if !IsDestDir(dest) {
		if !filepath.IsAbs(dest) {
			return filepath.Join(cwd, dest)
		}
		return dest
	}
	urlBase := filepath.Base(rawurl)
	destPath := filepath.Join(dest, urlBase)

	if !filepath.IsAbs(dest) {
		destPath = filepath.Join(cwd, destPath)
	}
	return destPath
}

func IsSrcsValid(srcsAndDest instructions.SourcesAndDest, resolvedSources []string, root string) error {
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	dest := srcsAndDest[len(srcsAndDest)-1]

	if !ContainsWildcards(srcs) {
		if len(srcs) > 1 && !IsDestDir(dest) {
			return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
		}
	}

	// If there is only one source and it's a directory, docker assumes the dest is a directory
	if len(resolvedSources) == 1 {
		if IsSrcRemoteFileURL(resolvedSources[0]) {
			return nil
		}
		fi, err := os.Lstat(filepath.Join(root, resolvedSources[0]))
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
	}

	totalFiles := 0
	for _, src := range resolvedSources {
		if IsSrcRemoteFileURL(src) {
			totalFiles++
			continue
		}
		src = filepath.Clean(src)
		files, err := RelativeFiles(src, root)
		if err != nil {
			return err
		}
		totalFiles += len(files)
	}
	if totalFiles == 0 {
		return errors.New("copy failed: no source files specified")
	}
	// If there are wildcards, and the destination is a file, there must be exactly one file to copy over,
	// Otherwise, return an error
	if !IsDestDir(dest) && totalFiles > 1 {
		return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
	}
	return nil
}

func IsSrcRemoteFileURL(rawurl string) bool {
	_, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return false
	}
	_, err = http.Get(rawurl)
	return err == nil
}

func UpdateConfigEnv(newEnvs []instructions.KeyValuePair, config *v1.Config, replacementEnvs []string) error {
	for index, pair := range newEnvs {
		expandedKey, err := ResolveEnvironmentReplacement(pair.Key, replacementEnvs, false)
		if err != nil {
			return err
		}
		expandedValue, err := ResolveEnvironmentReplacement(pair.Value, replacementEnvs, false)
		if err != nil {
			return err
		}
		newEnvs[index] = instructions.KeyValuePair{
			Key:   expandedKey,
			Value: expandedValue,
		}
	}

	// First, convert config.Env array to []instruction.KeyValuePair
	var kvps []instructions.KeyValuePair
	for _, env := range config.Env {
		entry := strings.SplitN(env, "=", 2)
		kvps = append(kvps, instructions.KeyValuePair{
			Key:   entry[0],
			Value: entry[1],
		})
	}
	// Iterate through new environment variables, and replace existing keys
	// We can't use a map because we need to preserve the order of the environment variables
Loop:
	for _, newEnv := range newEnvs {
		for index, kvp := range kvps {
			// If key exists, replace the KeyValuePair...
			if kvp.Key == newEnv.Key {
				logrus.Debugf("Replacing environment variable %v with %v in config", kvp, newEnv)
				kvps[index] = newEnv
				continue Loop
			}
		}
		// ... Else, append it as a new env variable
		kvps = append(kvps, newEnv)
	}
	// Convert back to array and set in config
	envArray := []string{}
	for _, kvp := range kvps {
		entry := kvp.Key + "=" + kvp.Value
		envArray = append(envArray, entry)
	}
	config.Env = envArray
	return nil
}

func GetUserFromUsername(userStr string, groupStr string) (string, string, error) {
	// Lookup by username
	userObj, err := user.Lookup(userStr)
	if err != nil {
		if _, ok := err.(user.UnknownUserError); ok {
			// Lookup by id
			userObj, err = user.LookupId(userStr)
			if err != nil {
				return "", "", err
			}
		} else {
			return "", "", err
		}
	}

	// Same dance with groups
	var group *user.Group
	if groupStr != "" {
		group, err = user.LookupGroup(groupStr)
		if err != nil {
			if _, ok := err.(user.UnknownGroupError); ok {
				group, err = user.LookupGroupId(groupStr)
				if err != nil {
					return "", "", err
				}
			} else {
				return "", "", err
			}
		}
	}

	uid := userObj.Uid
	gid := ""
	if group != nil {
		gid = group.Gid
	}

	return uid, gid, nil
}
