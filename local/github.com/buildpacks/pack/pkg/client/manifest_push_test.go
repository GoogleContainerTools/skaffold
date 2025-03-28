package client

import (
	"bytes"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPushManifest(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testPushManifest, spec.Report(report.Terminal{}))
}

func testPushManifest(t *testing.T, when spec.G, it spec.S) {
	var (
		mockController   *gomock.Controller
		mockIndexFactory *testmocks.MockIndexFactory
		out              bytes.Buffer
		logger           logging.Logger
		subject          *Client
		err              error
		tmpDir           string
	)
	it.Before(func() {
		logger = logging.NewLogWithWriters(&out, &out, logging.WithVerbose())
		mockController = gomock.NewController(t)
		mockIndexFactory = testmocks.NewMockIndexFactory(mockController)

		subject, err = NewClient(
			WithLogger(logger),
			WithIndexFactory(mockIndexFactory),
			WithExperimental(true),
			WithKeychain(authn.DefaultKeychain),
		)
		h.AssertSameInstance(t, mockIndexFactory, subject.indexFactory)
		h.AssertNil(t, err)
	})
	it.After(func() {
		mockController.Finish()
		h.AssertNil(t, os.RemoveAll(tmpDir))
	})

	when("#PushManifest", func() {
		when("index exists locally", func() {
			var index *h.MockImageIndex

			it.Before(func() {
				index = h.NewMockImageIndex(t, "some-index", 1, 2)
				mockIndexFactory.EXPECT().LoadIndex(gomock.Eq("some-index"), gomock.Any()).Return(index, nil)
			})
			it("pushes the index to the registry", func() {
				err = subject.PushManifest(PushManifestOptions{
					IndexRepoName: "some-index",
				})
				h.AssertNil(t, err)
				h.AssertTrue(t, index.PushCalled)
			})
		})

		when("index doesn't exist locally", func() {
			it.Before(func() {
				mockIndexFactory.EXPECT().LoadIndex(gomock.Any(), gomock.Any()).Return(nil, errors.New("ErrNoImageOrIndexFoundWithGivenDigest"))
			})

			it("errors with a message", func() {
				err = subject.PushManifest(PushManifestOptions{
					IndexRepoName: "some-index",
				})
				h.AssertNotNil(t, err)
			})
		})
	})
}
