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
	"strings"
	"testing"

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
