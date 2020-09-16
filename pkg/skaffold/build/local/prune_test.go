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

package local

import (
	"context"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDiskUsage(t *testing.T) {
	tests := []struct {
		ctxFunc             func() context.Context
		description         string
		fails               uint
		expectedUtilization uint64
		shouldErr           bool
	}{
		{
			description:         "happy path",
			fails:               0,
			shouldErr:           false,
			expectedUtilization: testutil.TestUtilization,
		},
		{
			description:         "first attempts failed",
			fails:               usageRetries - 1,
			shouldErr:           false,
			expectedUtilization: testutil.TestUtilization,
		},
		{
			description:         "all attempts failed",
			fails:               usageRetries,
			shouldErr:           true,
			expectedUtilization: 0,
		},
		{
			description:         "context cancelled",
			fails:               0,
			shouldErr:           true,
			expectedUtilization: 0,
			ctxFunc: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
				DUFails: test.fails,
			}), true)

			ctx := context.Background()
			if test.ctxFunc != nil {
				ctx = test.ctxFunc()
			}
			res, err := pruner.diskUsage(ctx)

			t.CheckError(test.shouldErr, err)
			if res != test.expectedUtilization {
				t.Errorf("invalid disk usage. got %d expected %d", res, test.expectedUtilization)
			}
		})
	}
}

/*
func (b *Builder) collectImagesToPrune(ctx context.Context, limit int, artifacts []*latest.Artifact) []string {
	imgNameCount := make(map[string]int)
	for _, a := range artifacts {
		imgNameCount[a.ImageName]++
	}
	rt := make([]string, 0)
	for _, a := range artifacts {
		imgs, err := b.listUniqImages(ctx, a.ImageName)
		if err != nil {
			logrus.Warnf("failed to list images: %v", err)
			continue
		}
		limForImage := limit * imgNameCount[a.ImageName]
		for i := limForImage; i < len(imgs); i++ {
			rt = append(rt, imgs[i].ID)
		}
	}
	return rt
}
*/
func TestCollectPruneImages(t *testing.T) {
	tests := []struct {
		description     string
		localImages     map[string][]string
		imagesToBuild   []string
		expectedToPrune []string
	}{
		{
			description: "todo",
			localImages: map[string][]string{
				"foo": {"111", "222", "333", "444"},
				"bar": {"555", "666", "777"},
			},
			imagesToBuild:   []string{"foo", "bar"},
			expectedToPrune: []string{"111", "222", "333", "555", "666"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
				LocalImages: test.localImages,
			}), true)

			res := pruner.collectImagesToPrune(
				context.Background(), artifacts(test.imagesToBuild...))
			sort.Strings(test.expectedToPrune)
			sort.Strings(res)
			t.CheckDeepEqual(res, test.expectedToPrune)
		})
	}
}
func artifacts(images ...string) []*latest.Artifact {
	rt := make([]*latest.Artifact, 0)
	for _, image := range images {
		rt = append(rt, a(image))
	}
	return rt
}

func a(name string) *latest.Artifact {
	return &latest.Artifact{
		ImageName: name,
		ArtifactType: latest.ArtifactType{
			DockerArtifact: &latest.DockerArtifact{},
		},
	}
}
