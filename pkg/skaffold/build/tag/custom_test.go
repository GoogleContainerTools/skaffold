/*
Copyright 2018 Google LLC

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

	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

func TestCustomTag_GenerateFullyQualifiedImageName(t *testing.T) {
	opts := &TagOptions{
		ImageName: "test",
		Digest:    "sha256:12345abcde",
	}

	expectedTag := "1.2.3-beta"

	c := &CustomTag{
		Tag: expectedTag,
	}
	tag, err := c.GenerateFullyQualifiedImageName(".", opts)
	testutil.CheckErrorAndDeepEqual(t, false, err, "test:"+expectedTag, tag)
}
