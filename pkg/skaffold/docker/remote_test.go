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

package docker

import (
	"strings"
	"testing"
)

func TestRemoteDigest(t *testing.T) {
	validReferences := []string{
		"python",
		"python:3-slim",
	}

	for _, ref := range validReferences {
		_, err := RemoteDigest(ref)

		// Ignore networking errors
		if err != nil && strings.Contains(err.Error(), "could not parse") {
			t.Errorf("unable to parse %q: %v", ref, err)
		}
	}
}
