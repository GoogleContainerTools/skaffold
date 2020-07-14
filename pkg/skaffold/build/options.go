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
	"strconv"
)

// Configuration denotes build configuration for the artifact builder as either Release or Debug
type Configuration int

const (
	Release = iota
	Debug
)

// ImageOptions provides options for the artifact builder
type ImageOptions struct {
	// FIXME: Tag should be []string but we don't support multiple tags yet
	Tag           string             // image tag
	Configuration Configuration      // build image for release or debug
	Args          map[string]*string // additional builder specific args
}

var CurrentConfiguration Configuration

func CreateBuilderOptions(tag string) *ImageOptions {
	return &ImageOptions{
		Tag:           tag,
		Configuration: CurrentConfiguration,
	}
}

// Hash returns the hash of given image option, useful for image caching
func (opts *ImageOptions) Hash() (string, error) {
	var inputs []string
	if opts == nil {
		return "", nil
	}

	inputs = append(inputs, strconv.Itoa(int(opts.Configuration)))
	if opts.Args != nil {
		inputs = append(inputs, convertBuildArgsToStringArray(opts.Args)...)
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
