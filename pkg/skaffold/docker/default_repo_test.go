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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestImageReplaceDefaultRepo(t *testing.T) {
	tests := []struct {
		description      string
		image            string
		defaultRepo      string
		expectedImage    string
		expectedImageNew string
		shouldErr        bool
	}{
		{
			description:      "basic GCR override",
			image:            "gcr.io/some/registry",
			defaultRepo:      "gcr.io/default",
			expectedImage:    "gcr.io/default/gcr.io/some/registry",
			expectedImageNew: "gcr.io/default/registry",
		},
		{
			description:      "no default repo set",
			image:            "gcr.io/some/registry",
			expectedImage:    "gcr.io/some/registry",
			expectedImageNew: "gcr.io/some/registry",
		},
		{
			description:      "prefix example on docs",
			image:            "gcr.io/k8s-skaffold/lib/core",
			defaultRepo:      "gcr.io/myproject/path/subpath",
			expectedImage:    "gcr.io/myproject/path/subpath/gcr.io/k8s-skaffold/lib/core",
			expectedImageNew: "gcr.io/myproject/path/subpath/lib/core",
		},
		{
			description:      "provided image has defaultRepo prefix",
			image:            "gcr.io/default/registry",
			defaultRepo:      "gcr.io/default",
			expectedImage:    "gcr.io/default/registry",
			expectedImageNew: "gcr.io/default/registry",
		},
		{
			description:      "provided image and defaultRepo have eu prefix",
			image:            "eu.gcr.io/project/registry",
			defaultRepo:      "eu.gcr.io/project",
			expectedImage:    "eu.gcr.io/project/registry",
			expectedImageNew: "eu.gcr.io/project/registry",
		},
		{
			description:      "default repo registry is in another domain and different subpaths",
			image:            "gcr.io/project1/subpath/registry",
			defaultRepo:      "eu.gcr.io/project2/defaultRepoSubpath",
			expectedImage:    "eu.gcr.io/project2/defaultRepoSubpath/gcr.io/project1/subpath/registry",
			expectedImageNew: "eu.gcr.io/project2/defaultRepoSubpath/subpath/registry",
		},
		{
			description:      "default repo registry and subset of same subpaths",
			image:            "gcr.io/project1/subpath/another/app1/registry",
			defaultRepo:      "eu.gcr.io/project2/subpath/another/dev",
			expectedImage:    "eu.gcr.io/project2/subpath/another/dev/gcr.io/project1/subpath/another/app1/registry",
			expectedImageNew: "eu.gcr.io/project2/subpath/another/dev/app1/registry",
		},
		{
			description:      "registry has shared prefix with defaultRepo",
			image:            "gcr.io/default/example/registry",
			defaultRepo:      "gcr.io/default/repository",
			expectedImage:    "gcr.io/default/repository/example/registry",
			expectedImageNew: "gcr.io/default/repository/example/registry",
		},
		{
			description:      "aws",
			image:            "gcr.io/some/registry",
			defaultRepo:      "aws_account_id.dkr.ecr.region.amazonaws.com",
			expectedImage:    "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_some_registry",
			expectedImageNew: "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_some_registry",
		},
		{
			description:      "aws over 255 chars",
			image:            "gcr.io/herewehaveanincrediblylongregistryname/herewealsohaveanabnormallylongimagename/doubtyouveseenanimagethislong/butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere/goodluckpushingthistoanyregistrymyfriend",
			defaultRepo:      "aws_account_id.dkr.ecr.region.amazonaws.com",
			expectedImage:    "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_herewehaveanincrediblylongregistryname_herewealsohaveanabnormallylongimagename_doubtyouveseenanimagethislong_butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere_goodluckpushingthistoanyregistrymyfrien",
			expectedImageNew: "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_herewehaveanincrediblylongregistryname_herewealsohaveanabnormallylongimagename_doubtyouveseenanimagethislong_butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere_goodluckpushingthistoanyregistrymyfrien",
		},
		{
			description:      "normal GCR concatenation with numbers and other characters",
			image:            "gcr.io/k8s-skaffold/skaffold-example",
			defaultRepo:      "gcr.io/k8s-skaffold",
			expectedImage:    "gcr.io/k8s-skaffold/skaffold-example",
			expectedImageNew: "gcr.io/k8s-skaffold/skaffold-example",
		},
		{
			description:      "keep tag",
			image:            "img:tag",
			defaultRepo:      "gcr.io/default",
			expectedImage:    "gcr.io/default/img:tag",
			expectedImageNew: "gcr.io/default/img:tag",
		},
		{
			description:      "keep digest",
			image:            "img@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			defaultRepo:      "gcr.io/default",
			expectedImage:    "gcr.io/default/img@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedImageNew: "gcr.io/default/img@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
		},
		{
			description:      "keep tag and digest",
			image:            "img:tag@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			defaultRepo:      "gcr.io/default",
			expectedImage:    "gcr.io/default/img:tag@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedImageNew: "gcr.io/default/img:tag@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
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
			replaced, err := substituteDefaultRepoIntoImage(test.defaultRepo, test.image)
			replacedNew, errNew := substituteDefaultRepoIntoImageNew(test.defaultRepo, test.image)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedImage, replaced)
			t.CheckErrorAndDeepEqual(test.shouldErr, errNew, test.expectedImageNew, replacedNew)
		})
	}
}
