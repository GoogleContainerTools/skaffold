/*
Copyright 2021 The Skaffold Authors

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

package v2

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

func TestDebuggingContainer(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	found := func() bool {
		for _, dc := range handler.getState().DebuggingContainers {
			if dc.Namespace == "ns" && dc.PodName == "pod" && dc.ContainerName == "container" {
				return true
			}
		}
		return false
	}
	notFound := func() bool { return !found() }
	wait(t, notFound)
	DebuggingContainerStarted("pod", "container", "ns", "artifact", "runtime", "/", nil)
	wait(t, found)
	DebuggingContainerTerminated("pod", "container", "ns", "artifact", "runtime", "/", nil)
	wait(t, notFound)
}
