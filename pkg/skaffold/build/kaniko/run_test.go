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

package kaniko

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
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
		expectedArgs []string
	}{
		{
			name:         "no build args",
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args"},
		}, {
			name: "buid args",
			buildArgs: map[string]*string{
				"nil_key":   nil,
				"empty_key": pointer(""),
				"value_key": pointer("value"),
			},
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args", "--build-arg", "empty_key=", "--build-arg", "nil_key", "--build-arg", "value_key=value"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := appendBuildArgsIfExists(test.args, test.buildArgs)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedArgs, actual)
		})
	}
}

func TestAppendInsecureRegistriesIfExist(t *testing.T) {
	insecureRegistry1 := fmt.Sprintf("insecure1-%d.registry.com", rand.Int())
	insecureRegistry2 := fmt.Sprintf("insecure2-%d.registry.com", rand.Int())
	notInsecureRegistry := fmt.Sprintf("not-insecure-%d.registry.com", rand.Int())
	tests := []struct {
		name               string
		insecureRegistries map[string]bool
		args               []string
		expectedArgs       []string
	}{
		{
			name:         "no insecure registries args",
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args"},
		}, {
			name: "insecure registries not empty but not none insecure",
			insecureRegistries: map[string]bool{
				notInsecureRegistry: false,
			},
			args:         []string{"first", "args"},
			expectedArgs: []string{"first", "args"},
		}, {
			name: "insecure registries with some insecure",
			insecureRegistries: map[string]bool{
				insecureRegistry1:   true,
				notInsecureRegistry: false,
				insecureRegistry2:   true,
			},
			args: []string{"first", "args"},
			// TODO: order should be deterministic in go 1.12+? check test isn't brittle
			expectedArgs: []string{"first", "args", "--insecure-registry", insecureRegistry1, "--insecure-registry", insecureRegistry2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := appendInsecureRegistriesIfExist(test.args, test.insecureRegistries)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedArgs, actual)
		})
	}
}

func pointer(a string) *string {
	return &a
}
