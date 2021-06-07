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

package build

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

// TestMergeWithPreviousBuilds tests that artifacts are always kept in the same order
func TestMergeWithPreviousBuilds(t *testing.T) {
	builds := MergeWithPreviousBuilds([]graph.Artifact{artifact("img1", "tag1_1"), artifact("img2", "tag2_1")}, nil)
	testutil.CheckDeepEqual(t, "img1:tag1_1,img2:tag2_1", tags(builds))

	builds = MergeWithPreviousBuilds([]graph.Artifact{artifact("img1", "tag1_2")}, builds)
	testutil.CheckDeepEqual(t, "img1:tag1_2,img2:tag2_1", tags(builds))

	builds = MergeWithPreviousBuilds([]graph.Artifact{artifact("img2", "tag2_2")}, builds)
	testutil.CheckDeepEqual(t, "img1:tag1_2,img2:tag2_2", tags(builds))

	builds = MergeWithPreviousBuilds([]graph.Artifact{artifact("img1", "tag1_3"), artifact("img2", "tag2_3")}, builds)
	testutil.CheckDeepEqual(t, "img1:tag1_3,img2:tag2_3", tags(builds))
}

func TestTagWithDigest(t *testing.T) {
	tag := TagWithDigest("some-tag", "sha256:abcd1234")
	testutil.CheckDeepEqual(t, "some-tag@sha256:abcd1234", tag)
	tag = TagWithDigest("some-tag@sha256:abcd1234", "sha256:abcd1234")
	testutil.CheckDeepEqual(t, "some-tag@sha256:abcd1234", tag)
}

func artifact(image, tag string) graph.Artifact {
	return graph.Artifact{
		ImageName: image,
		Tag:       tag,
	}
}

func tags(artifacts []graph.Artifact) string {
	var tags string

	for i, a := range artifacts {
		if i > 0 {
			tags += ","
		}
		tags += fmt.Sprintf("%s:%s", a.ImageName, a.Tag)
	}

	return tags
}
