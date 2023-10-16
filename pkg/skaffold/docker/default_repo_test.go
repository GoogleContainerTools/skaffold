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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestImageReplaceDefaultRepo(t *testing.T) {
	tests := []struct {
		description    string
		image          string
		defaultRepo    string
		multiLevelRepo *bool
		expectedImage  string
		shouldErr      bool
	}{
		{
			description:   "basic GCR concatenation",
			image:         "gcr.io/some/registry",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/gcr.io/some/registry",
		},
		{
			description:    "basic GCR concatenation without multi-level",
			image:          "gcr.io/some/registry",
			defaultRepo:    "gcr.io/default",
			multiLevelRepo: util.Ptr(false),
			expectedImage:  "gcr.io/default/gcr.io/some/registry",
		},
		{
			description:   "basic AR concatenation",
			image:         "github.com/org/app",
			defaultRepo:   "us-central1-docker.pkg.dev/default",
			expectedImage: "us-central1-docker.pkg.dev/default/github.com/org/app",
		},
		{
			description:    "basic AR concatenation without multi-level",
			image:          "github.com/org/app",
			defaultRepo:    "us-central1-docker.pkg.dev/default",
			multiLevelRepo: util.Ptr(false),
			expectedImage:  "us-central1-docker.pkg.dev/default/github.com/org/app",
		},
		{
			description:   "no default repo set",
			image:         "gcr.io/some/registry",
			expectedImage: "gcr.io/some/registry",
		},
		{
			description:   "provided image has defaultRepo prefix",
			image:         "gcr.io/default/registry",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/registry",
		},
		{
			description:   "provided image has defaultRepo eu prefix",
			image:         "eu.gcr.io/project/registry",
			defaultRepo:   "eu.gcr.io/project",
			expectedImage: "eu.gcr.io/project/registry",
		},
		{
			description:   "image has shared prefix with defaultRepo",
			image:         "gcr.io/default/example/registry",
			defaultRepo:   "gcr.io/default/repository",
			expectedImage: "gcr.io/default/repository/example/registry",
		},
		{
			description:   "aws",
			image:         "gcr.io/some/registry",
			defaultRepo:   "aws_account_id.dkr.ecr.region.amazonaws.com",
			expectedImage: "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_some_registry",
		},
		{
			description:    "aws multi-level",
			image:          "gcr.io/some/registry",
			defaultRepo:    "aws_account_id.dkr.ecr.region.amazonaws.com",
			multiLevelRepo: util.Ptr(true),
			expectedImage:  "aws_account_id.dkr.ecr.region.amazonaws.com/gcr.io/some/registry",
		},
		{
			description:   "aws over 255 chars",
			image:         "gcr.io/herewehaveanincrediblylongregistryname/herewealsohaveanabnormallylongimagename/doubtyouveseenanimagethislong/butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere/goodluckpushingthistoanyregistrymyfriend",
			defaultRepo:   "aws_account_id.dkr.ecr.region.amazonaws.com",
			expectedImage: "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_herewehaveanincrediblylongregistryname_herewealsohaveanabnormallylongimagename_doubtyouveseenanimagethislong_butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere_goodluckpushingthistoanyregistrymyfrien",
		},
		{
			description:   "normal GCR concatenation with numbers and other characters",
			image:         "gcr.io/k8s-skaffold/skaffold-example",
			defaultRepo:   "gcr.io/k8s-skaffold",
			expectedImage: "gcr.io/k8s-skaffold/skaffold-example",
		},
		{
			description:   "image has shared prefix with defaultRepo, but not gcr",
			image:         "myrepo/skaffold-example",
			defaultRepo:   "myrepo",
			expectedImage: "myrepo/skaffold-example",
		},
		{
			description:   "ko with GCR",
			image:         "ko://example.com/cmd/app",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/example.com/cmd/app",
		},
		{
			description:   "ko not with GCR",
			image:         "ko://example.com/cmd/app",
			defaultRepo:   "myrepo",
			expectedImage: "myrepo/example_com_cmd_app",
		},
		{
			description:    "ko multi-level not with GCR",
			image:          "ko://example.com/cmd/app",
			defaultRepo:    "myrepo",
			multiLevelRepo: util.Ptr(true),
			expectedImage:  "myrepo/example.com/cmd/app",
		},
		{
			description:   "keep tag",
			image:         "img:tag",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/img:tag",
		},
		{
			description:   "keep digest",
			image:         "img@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/img@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
		},
		{
			description:   "keep tag and digest",
			image:         "img:tag@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/img:tag@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
		},
		{
			description: "invalid",
			defaultRepo: "gcr.io/default",
			image:       "!!invalid!!",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			replaced, err := SubstituteDefaultRepoIntoImage(test.defaultRepo, test.multiLevelRepo, test.image)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedImage, replaced)
		})
	}
}
