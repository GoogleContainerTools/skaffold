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

	packcfg "github.com/buildpacks/pack/config"
	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFetcher(t *testing.T) {
	tests := []struct {
		description    string
		imageExists    bool
		pull           packcfg.PullPolicy
		expectedPulled []string
	}{
		{
			description:    "pull",
			pull:           packcfg.PullAlways,
			expectedPulled: []string{"image"},
		},
		{
			description:    "pull-if-not-present but image present",
			pull:           packcfg.PullIfNotPresent,
			imageExists:    true,
			expectedPulled: nil,
		},
		{
			description:    "pull-if-not-present",
			pull:           packcfg.PullIfNotPresent,
			expectedPulled: []string{"image"},
		},
		{
			description:    "don't pull",
			pull:           packcfg.PullNever,
			expectedPulled: nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			api := &testutil.FakeAPIClient{}
			if test.imageExists {
				api.Add("image", "sha256:deadbeef")
			}
			docker := fakeLocalDaemon(api)

			var out bytes.Buffer

			f := newFetcher(&out, docker)
			f.Fetch(context.Background(), "image", true, test.pull)

			t.CheckDeepEqual(test.expectedPulled, api.Pulled())
		})
	}
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
}
