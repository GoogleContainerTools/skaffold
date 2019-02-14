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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConfigureJvmDebugging(t *testing.T) {
	tests := []struct {
		description   string
		containerSpec map[interface{}]interface{}
		configuration imageConfiguration
		result        map[interface{}]interface{}
	}{
		{
			description:   "empty",
			containerSpec: map[interface{}]interface{}{},
			configuration: imageConfiguration{},
			result: map[interface{}]interface{}{
				"env": []interface{}{
					map[interface{}]interface{}{"name": "JAVA_TOOL_OPTIONS", "value": "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"},
				},
				"ports": []interface{}{
					map[interface{}]interface{}{"name": "jdwp", "containerPort": 5005},
				},
			},
		},
		{
			description:   "existing port",
			containerSpec: map[interface{}]interface{}{
				"ports": []interface{}{
					map[interface{}]interface{}{"name": "http-server", "containerPort": 8080},
				},
			},
			configuration: imageConfiguration{},
			result: map[interface{}]interface{}{
				"env": []interface{}{
					map[interface{}]interface{}{"name": "JAVA_TOOL_OPTIONS", "value": "-agentlib:jdwp=transport=dt_socket,server=y,address=5005,suspend=n,quiet=y"},
				},
				"ports": []interface{}{
					map[interface{}]interface{}{"name": "http-server", "containerPort": 8080},
					map[interface{}]interface{}{"name": "jdwp", "containerPort": 5005},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			configureJvmDebugging(test.containerSpec, test.configuration)
			testutil.CheckDeepEqual(t, test.result, test.containerSpec)
		})
	}
}
