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

package integration

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestBuildGCBWithExplicitRepo(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	// Other integration tests run with the --default-repo option.
	// This one explicitly specifies the full image name.
	skaffold.Build().WithRepo("").InDir("testdata/gcb-explicit-repo").RunOrFail(t)
}
