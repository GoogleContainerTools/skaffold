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
