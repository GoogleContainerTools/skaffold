/*
Copyright 2020 The Skaffold Authors

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

package buildpacks

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFetcher(t *testing.T) {
	tests := []struct {
		description    string
		pull           bool
		expectedPulled []string
	}{
		{
			description:    "pull",
			pull:           true,
			expectedPulled: []string{"image"},
		},
		{
			description:    "don't pull",
			pull:           false,
			expectedPulled: nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			api := &testutil.FakeAPIClient{}
			docker := docker.NewLocalDaemon(api, nil, false, nil)

			var out bytes.Buffer

			f := newFetcher(&out, docker)
			f.Fetch(context.Background(), "image", true, test.pull)

			t.CheckDeepEqual(test.expectedPulled, api.Pulled)
		})
	}
}
