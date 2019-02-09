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

package kubectl

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTraverse(t *testing.T) {
	manifest := map[interface{}]interface{}{
		"string": "value1",
		"node1": map[interface{}]interface{}{
			"node2": "value2",
		},
	}

	tests := []struct {
		description string
		path        []string
		found       bool
		result      interface{}
	}{
		{"1-level found string", []string{"string"}, true, "value1"},
		{"1-level found map", []string{"node1"}, true, manifest["node1"]},
		{"1-level not found", []string{"notfound"}, false, nil},
		{"2-level found string", []string{"node1", "node2"}, true, "value2"},
		{"2-level not found", []string{"node1", "notfound"}, false, nil},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result, found := traverse(manifest, test.path...)
			testutil.CheckDeepEqual(t, test.found, found)
			if test.found {
				testutil.CheckDeepEqual(t, test.result, result)
			}
		})
	}
}

func TestTraverseToMap(t *testing.T) {
	tests := []struct {
		description string
		source      map[interface{}]interface{}
		path        []string
		result      map[interface{}]interface{}
	}{
		{description: "existing map",
			source: map[interface{}]interface{}{
				"node1": map[interface{}]interface{}{
					"node2": "value2",
				}},
			path: []string{"node1"},
			result: map[interface{}]interface{}{
				"node1": map[interface{}]interface{}{
					"node2": "value2",
					"a":     "b",
				},
			},
		},
		{description: "non-map",
			source: map[interface{}]interface{}{"node1": "value1"},
			path:   []string{"node1"},
			result: map[interface{}]interface{}{
				"node1": map[interface{}]interface{}{"a": "b"},
			},
		},
		{description: "non-existant 1-level",
			source: map[interface{}]interface{}{},
			path:   []string{"node1"},
			result: map[interface{}]interface{}{
				"node1": map[interface{}]interface{}{"a": "b"},
			},
		},
		{description: "non-existant 2-level",
			source: map[interface{}]interface{}{},
			path:   []string{"node1", "node2"},
			result: map[interface{}]interface{}{
				"node1": map[interface{}]interface{}{
					"node2": map[interface{}]interface{}{"a": "b"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// retrieve from the source map and make a change
			result := traverseToMap(test.source, test.path...)
			result["a"] = "b"
			testutil.CheckDeepEqual(t, test.result, test.source)
		})
	}
}
func TestEnvAsMap(t *testing.T) {
	tests := []struct {
		description string
		source      []string
		result      map[string]string
	}{
		{description: "empty",
			source: []string{},
			result: map[string]string{},
		},
		{description: "single item",
			source: []string{"foo=bar"},
			result: map[string]string{"foo": "bar"},
		},
		{description: "multiple items",
			source: []string{"foo=bar", "baz=bop"},
			result: map[string]string{"foo": "bar", "baz": "bop"},
		},
		{description: "overridden key",
			source: []string{"foo=bar", "foo=baz"},
			result: map[string]string{"foo": "baz"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// retrieve from the source map and make a change
			result := envAsMap(test.source)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}

func TestFindArtifact(t *testing.T) {
	buildArtifacts := []build.Artifact{
		{ImageName: "image1", Tag: "tag1"},
	}
	tests := []struct {
		description string
		source      string
		returnNil   bool
	}{
		{description: "found",
			source:    "image1",
			returnNil: false,
		},
		{description: "not found",
			source:    "image2",
			returnNil: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := findArtifact(test.source, buildArtifacts)
			testutil.CheckDeepEqual(t, test.returnNil, result == nil)
		})
	}
}

func TestGuessRuntime(t *testing.T) {
	tests := []struct {
		description string
		source      imageConfiguration
		result      string
	}{
		{
			description: "JAVA_TOOL_OPTIONS",
			source:      imageConfiguration{env: map[string]string{"JAVA_TOOL_OPTIONS": "-agent:jdwp"}},
			result:      JVM,
		},
		{
			description: "JAVA_VERSION",
			source:      imageConfiguration{env: map[string]string{"JAVA_VERSION": "8"}},
			result:      JVM,
		},
		{
			description: "entrypoint java",
			source:      imageConfiguration{entrypoint: []string{"java", "-jar", "foo.jar"}},
			result:      JVM,
		},
		{
			description: "entrypoint /usr/bin/java",
			source:      imageConfiguration{entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      JVM,
		},
		{
			description: "no entrypoint, args java",
			source:      imageConfiguration{arguments: []string{"java", "-jar", "foo.jar"}},
			result:      JVM,
		},
		{
			description: "no entrypoint, arguments /usr/bin/java",
			source:      imageConfiguration{arguments: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      JVM,
		},
		{
			description: "other entrypoint, arguments /usr/bin/java",
			source:      imageConfiguration{entrypoint: []string{"/bin/sh"}, arguments: []string{"/usr/bin/java", "-jar", "foo.jar"}},
			result:      UNKNOWN,
		},
		{
			description: "entrypoint /bin/sh",
			source:      imageConfiguration{entrypoint: []string{"/bin/sh"}},
			result:      UNKNOWN,
		},
		{
			description: "nothing",
			source:      imageConfiguration{},
			result:      UNKNOWN,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// retrieve from the source map and make a change
			result := guessRuntime(test.source)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}
