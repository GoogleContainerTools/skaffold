package image

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestImageReplaceDefaultRepo(t *testing.T) {
	tests := []struct {
		description   string
		repo          string
		image         string
		defaultRepo   string
		expectedImage string
	}{
		{
			description:   "basic GCR concatenation",
			repo:	         "gcr.io/some",
			image:         "registry",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/gcr.io/some/registry",
		},
		{
			description:   "no default repo set",
			repo:          "gcr.io/some",
			image:         "registry",
			expectedImage: "gcr.io/some/registry",
		},
		{
			description:   "provided image has defaultRepo prefix",
			repo:          "gcr.io/default",
			image:         "registry",
			defaultRepo:   "gcr.io/default",
			expectedImage: "gcr.io/default/registry",
		},
		{
			description:   "image has shared prefix with defaultRepo",
			repo:          "gcr.io/default/example",
			image:         "registry",
			defaultRepo:   "gcr.io/default/repository",
			expectedImage: "gcr.io/default/repository/example/registry",
		},
		{
			description:   "aws",
			repo:          "gcr.io/some",
			image:         "registry",
			defaultRepo:   "aws_account_id.dkr.ecr.region.amazonaws.com",
			expectedImage: "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_some_registry",
		},
		{
			description:   "aws over 255 chars",
			repo:          "gcr.io/herewehaveanincrediblylongregistryname/herewealsohaveanabnormallylongimagename/doubtyouveseenanimagethislong/butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere",
			image:         "goodluckpushingthistoanyregistrymyfriend",
			defaultRepo:   "aws_account_id.dkr.ecr.region.amazonaws.com",
			expectedImage: "aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_herewehaveanincrediblylongregistryname_herewealsohaveanabnormallylongimagename_doubtyouveseenanimagethislong_butyouneverknowdoyouimeanpeopledosomecrazystuffoutthere_goodluckpushingthistoanyregistrymyfrien",
		},
		{
			description:   "normal GCR concatenation with numbers and other characters",
			repo:          "gcr.io/k8s-skaffold",
			image:         "skaffold-example",
			defaultRepo:   "gcr.io/k8s-skaffold",
			expectedImage: "gcr.io/k8s-skaffold/skaffold-example",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testReg := NewGenericContainerRegistry(test.repo)
			defaultReg := NewGenericContainerRegistry(test.defaultRepo)
			testImage := NewGenericImage(testReg, test.image)
			t.CheckDeepEqual(test.expectedImage, testImage.Update(defaultReg))
		})
	}
}
