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

package cluster

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestAppendCacheIfExists(t *testing.T) {
	tests := []struct {
		name         string
		cache        *latest.KanikoCache
		args         []string
		expectedArgs []string
	}{
		{
			name:         "no cache",
			cache:        nil,
			args:         []string{"some", "args"},
			expectedArgs: []string{"some", "args"},
		}, {
			name:         "cache layers",
			cache:        &latest.KanikoCache{},
			args:         []string{"some", "more", "args"},
			expectedArgs: []string{"some", "more", "args", "--cache=true"},
		}, {
			name: "cache layers to specific repo",
			cache: &latest.KanikoCache{
				Repo: "myrepo",
			},
			args:         []string{"initial", "args"},
			expectedArgs: []string{"initial", "args", "--cache=true", "--cache-repo=myrepo"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := appendCacheIfExists(test.args, test.cache)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedArgs, actual)
		})
	}
}

func TestAppendTargetIfExists(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		args         []string
		expectedArgs []string
	}{
		{
			name:         "pass in empty target",
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args"},
		}, {
			name:         "pass in target",
			target:       "stageOne",
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args", "--target=stageOne"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := appendTargetIfExists(test.args, test.target)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedArgs, actual)
		})
	}
}

func TestAppendBuildArgsIfExists(t *testing.T) {
	tests := []struct {
		name         string
		buildArgs    map[string]*string
		args         []string
		env          []string
		expectedArgs []string
		shouldErr    bool
	}{
		{
			name:         "no build args",
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args"},
		}, {
			name: "build args",
			buildArgs: map[string]*string{
				"nil_key":   nil,
				"empty_key": pointer(""),
				"value_key": pointer("value"),
			},
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args", "--build-arg", "empty_key=", "--build-arg", "nil_key", "--build-arg", "value_key=value"},
		}, {
			name: "build arg with env",
			buildArgs: map[string]*string{
				"value_key": pointer("value"),
				"env_key":   pointer("{{.KEY}}"),
			},
			args:         []string{"first", "args"},
			env:          []string{"KEY=VALUE"},
			expectedArgs: []string{"first", "args", "--build-arg", "env_key=VALUE", "--build-arg", "value_key=value"},
		}, {
			name: "build arg with bad template",
			buildArgs: map[string]*string{
				"value_key":    pointer("value"),
				"template_key": pointer("{{.BAD_TEMPLATE}"),
			},
			args:      []string{"first", "args"},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		util.OSEnviron = func() []string {
			return test.env
		}
		t.Run(test.name, func(t *testing.T) {
			actual, err := appendBuildArgsIfExists(test.args, test.buildArgs)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedArgs, actual)
		})
	}
}

func pointer(a string) *string {
	return &a
}
