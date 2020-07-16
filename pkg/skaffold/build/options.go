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

package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Configuration denotes build configuration for the artifact builder as either Dev or Debug
type Configuration string

const (
	Dev   = Configuration("dev")
	Debug = Configuration("debug")
)

// BuilderOptions provides options for the artifact builder
type BuilderOptions struct {
	Tag           string             // image tag
	Configuration Configuration      // build image for dev or debug
	BuildArgs     map[string]*string // additional builder specific args
}

// Hash returns the hash of given image option, useful for image caching
func (opts *BuilderOptions) Hash() (string, error) {
	var inputs []string
	if opts == nil {
		return "", nil
	}

	inputs = append(inputs, string(opts.Configuration))
	if opts.BuildArgs != nil {
		inputs = append(inputs, convertBuildArgsToStringArray(opts.BuildArgs)...)
	}

	hasher := sha256.New()
	enc := json.NewEncoder(hasher)
	if err := enc.Encode(inputs); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func convertBuildArgsToStringArray(buildArgs map[string]*string) []string {
	var args []string
	for k, v := range buildArgs {
		if v == nil {
			args = append(args, k)
			continue
		}
		args = append(args, fmt.Sprintf("%s=%s", k, *v))
	}
	sort.Strings(args)
	return args
}
