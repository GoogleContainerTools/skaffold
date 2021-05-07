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

package tag

import (
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSha256_GenerateTag(t *testing.T) {
	c := &ChecksumTagger{}

	image := latestV1.Artifact{
		ImageName: "img:tag",
	}

	tag, err := c.GenerateTag(image)
	testutil.CheckErrorAndDeepEqual(t, false, err, "", tag)

	image.ImageName = "img"
	tag, err = c.GenerateTag(image)
	testutil.CheckErrorAndDeepEqual(t, false, err, "latest", tag)

	image.ImageName = "registry.example.com:8080/img:tag"
	tag, err = c.GenerateTag(image)
	testutil.CheckErrorAndDeepEqual(t, false, err, "", tag)

	image.ImageName = "registry.example.com:8080/img"
	tag, err = c.GenerateTag(image)
	testutil.CheckErrorAndDeepEqual(t, false, err, "latest", tag)

	image.ImageName = "registry.example.com/img"
	tag, err = c.GenerateTag(image)
	testutil.CheckErrorAndDeepEqual(t, false, err, "latest", tag)

	image.ImageName = "registry.example.com:8080:garbage"
	tag, err = c.GenerateTag(image)
	testutil.CheckErrorAndDeepEqual(t, true, err, "", tag)
}
