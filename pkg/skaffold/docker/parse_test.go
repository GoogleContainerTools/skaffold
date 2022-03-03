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

package docker

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/moby/buildkit/frontend/dockerfile/parser"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUnquote(t *testing.T) {
	testutil.CheckDeepEqual(t, `scratch`, unquote(`scratch`))
	testutil.CheckDeepEqual(t, `scratch`, unquote(`"scratch"`))
	testutil.CheckDeepEqual(t, `scratch`, unquote(`""scratch""`))
	testutil.CheckDeepEqual(t, `scratch`, unquote(`'scratch'`))
	testutil.CheckDeepEqual(t, `scratch`, unquote(`''scratch''`))
	testutil.CheckDeepEqual(t, `'scratch'`, unquote(`"'scratch'"`))
	testutil.CheckDeepEqual(t, `golang:1.15`, unquote(`golang:"1.15"`))
	testutil.CheckDeepEqual(t, `golang:1.15`, unquote(`golang:'1.15'`))
}

func TestRemoveExtraBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		dockerfile  string
		buildArgs   map[string]*string
		expected    map[string]*string
	}{
		{
			description: "no args in dockerfile",
			dockerfile:  `FROM nginx:stable`,
			buildArgs: map[string]*string{
				"foo": util.StringPtr("FOO"),
				"bar": util.StringPtr("BAR"),
			},
			expected: map[string]*string{},
		},
		{
			description: "exact args in dockerfile",
			dockerfile: `ARG foo
ARG bar
FROM nginx:stable`,
			buildArgs: map[string]*string{
				"foo": util.StringPtr("FOO"),
				"bar": util.StringPtr("BAR"),
			},
			expected: map[string]*string{
				"foo": util.StringPtr("FOO"),
				"bar": util.StringPtr("BAR"),
			},
		},
		{
			description: "extra build args",
			dockerfile: `ARG foo
ARG bar
FROM nginx:stable`,
			buildArgs: map[string]*string{
				"foo":    util.StringPtr("FOO"),
				"bar":    util.StringPtr("BAR"),
				"foobar": util.StringPtr("FOOBAR"),
				"gopher": util.StringPtr("GOPHER"),
			},
			expected: map[string]*string{
				"foo": util.StringPtr("FOO"),
				"bar": util.StringPtr("BAR"),
			},
		},
		{
			description: "extra build args for multistage",
			dockerfile: `ARG foo
FROM nginx:stable
ARG bar1
FROM golang:stable
ARG bar2`,
			buildArgs: map[string]*string{
				"foo":  util.StringPtr("FOO"),
				"bar1": util.StringPtr("BAR"),
				"bar2": util.StringPtr("BAR2"),
			},
			expected: map[string]*string{
				"foo":  util.StringPtr("FOO"),
				"bar1": util.StringPtr("BAR"),
				"bar2": util.StringPtr("BAR2"),
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			r := strings.NewReader(test.dockerfile)
			got, _ := filterUnusedBuildArgs(r, test.buildArgs)
			t.CheckDeepEqual(test.expected, got)
		})
	}
}

func TestValidateParsedDockerfile(t *testing.T) {
	tests := []struct {
		description string
		dockerfile  string
		shouldErr   bool
	}{
		{
			description: "valid Dockerfile",
			dockerfile:  `FROM foo`,
		},
		{
			description: "invalid Dockerfile",
			dockerfile:  `BAR foo`,
			shouldErr:   true,
		},
		{
			description: "explicit syntax directive",
			dockerfile: `# syntax = foo/bar

			[package]
			name = "foo"
			version = "0.1.0"`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			res, err := parser.Parse(bytes.NewReader([]byte(test.dockerfile)))
			t.CheckNoError(err)
			err = validateParsedDockerfile(bytes.NewReader([]byte(test.dockerfile)), res)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestIsOldImageManifestProblem(t *testing.T) {
	tests := []struct {
		description string
		mode        config.RunMode
		err         error
		expectedMsg string
		expected    bool
	}{
		{
			description: "dev command older manifest with image name",
			mode:        config.RunModes.Dev,
			err:         fmt.Errorf(`listing files: parsing ONBUILD instructions: retrieving image "library/ruby:2.3.0": unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expectedMsg: "Could not retrieve image library/ruby:2.3.0 pushed with the deprecated manifest v1. Ignoring files dependencies for all ONBUILD triggers. To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry.",
			expected:    true,
		},
		{
			description: "dev command older manifest without image name",
			mode:        config.RunModes.Dev,
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expectedMsg: "Could not retrieve image pushed with the deprecated manifest v1. Ignoring files dependencies for all ONBUILD triggers. To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry.",
			expected:    true,
		},
		{
			description: "dev command with random name",
			mode:        config.RunModes.Dev,
			err:         fmt.Errorf(`listing files: parsing ONBUILD instructions: retrieve image "noimage" image does not exits`),
		},
		{
			description: "debug command older manifest",
			mode:        config.RunModes.Debug,
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expectedMsg: "Could not retrieve image pushed with the deprecated manifest v1. Ignoring files dependencies for all ONBUILD triggers. To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry.",
			expected:    true,
		},
		{
			description: "build command older manifest",
			mode:        config.RunModes.Build,
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expected:    true,
		},
		{
			description: "run command older manifest",
			mode:        config.RunModes.Run,
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expected:    true,
		},
		{
			description: "deploy command older manifest",
			mode:        config.RunModes.Deploy,
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := mockConfig{mode: test.mode}
			actualMsg, actual, _ := isOldImageManifestProblem(cfg, test.err)
			t.CheckDeepEqual(test.expectedMsg, actualMsg)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
