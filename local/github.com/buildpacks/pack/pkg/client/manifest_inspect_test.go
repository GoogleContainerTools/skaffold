package client

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/buildpacks/imgutil"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestInspectManifest(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testInspectManifest, spec.Report(report.Terminal{}))
}

func testInspectManifest(t *testing.T, when spec.G, it spec.S) {
	var (
		mockController   *gomock.Controller
		mockIndexFactory *testmocks.MockIndexFactory
		stdout           bytes.Buffer
		stderr           bytes.Buffer
		logger           logging.Logger
		subject          *Client
		err              error
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&stdout, &stderr, logging.WithVerbose())
		mockController = gomock.NewController(t)
		mockIndexFactory = testmocks.NewMockIndexFactory(mockController)

		subject, err = NewClient(
			WithLogger(logger),
			WithIndexFactory(mockIndexFactory),
			WithExperimental(true),
			WithKeychain(authn.DefaultKeychain),
		)
		h.AssertSameInstance(t, mockIndexFactory, subject.indexFactory)
		h.AssertSameInstance(t, subject.logger, logger)
		h.AssertNil(t, err)
	})
	it.After(func() {
		mockController.Finish()
	})

	when("#InspectManifest", func() {
		var indexRepoName string

		when("index doesn't exits", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				mockIndexFactory.
					EXPECT().
					FindIndex(gomock.Eq(indexRepoName), gomock.Any()).Return(nil, errors.New("index not found"))
			})

			it("should return an error when index not found", func() {
				err = subject.InspectManifest(indexRepoName)
				h.AssertEq(t, err.Error(), "index not found")
			})
		})

		when("index exists", func() {
			var indexManifest *v1.IndexManifest

			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				idx := setUpIndex(t, indexRepoName, *mockIndexFactory)
				indexManifest, err = idx.IndexManifest()
				h.AssertNil(t, err)
			})

			it("should return formatted IndexManifest", func() {
				err = subject.InspectManifest(indexRepoName)
				h.AssertNil(t, err)

				printedIndex := &v1.IndexManifest{}
				err = json.Unmarshal(stdout.Bytes(), printedIndex)
				h.AssertEq(t, indexManifest, printedIndex)
			})
		})
	})
}

func setUpIndex(t *testing.T, indexRepoName string, mockIndexFactory testmocks.MockIndexFactory) v1.ImageIndex {
	randomUnderlyingIndex, err := random.Index(1024, 1, 2)
	h.AssertNil(t, err)

	options := &imgutil.IndexOptions{
		BaseIndex: randomUnderlyingIndex,
	}
	idx, err := imgutil.NewCNBIndex(indexRepoName, *options)
	h.AssertNil(t, err)

	mockIndexFactory.EXPECT().FindIndex(gomock.Eq(indexRepoName), gomock.Any()).Return(idx, nil)
	return randomUnderlyingIndex
}
