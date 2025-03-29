package client

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil"
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

func TestDeleteManifest(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testDeleteManifest, spec.Report(report.Terminal{}))
}

func testDeleteManifest(t *testing.T, when spec.G, it spec.S) {
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

		tmpDir, err = os.MkdirTemp("", "remove-manifest-test")
		h.AssertNil(t, err)
		os.Setenv("XDG_RUNTIME_DIR", tmpDir)

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

	when("#DeleteManifest", func() {
		var (
			indexPath     string
			indexRepoName string
		)

		when("index doesn't exists", func() {
			it.Before(func() {
				mockIndexFactory.EXPECT().LoadIndex(gomock.Any(), gomock.Any()).Return(nil, errors.New("index not found locally"))
			})
			it("should return an error when index is already deleted", func() {
				err = subject.DeleteManifest([]string{"pack/none-existent-index"})
				h.AssertNotNil(t, err)
			})
		})

		when("index exists", func() {
			var idx imgutil.ImageIndex

			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				indexPath = filepath.Join(tmpDir, imgutil.MakeFileSafeName(indexRepoName))
				idx = h.RandomCNBIndex(t, indexRepoName, 1, 1)
				mockIndexFactory.EXPECT().LoadIndex(gomock.Eq(indexRepoName), gomock.Any()).Return(idx, nil)

				// Let's write the index on disk
				h.AssertNil(t, idx.SaveDir())
			})

			it("should delete local index", func() {
				err = subject.DeleteManifest([]string{indexRepoName})
				h.AssertNil(t, err)
				h.AssertContains(t, out.String(), "Successfully deleted manifest list(s) from local storage")
				h.AssertPathDoesNotExists(t, indexPath)
			})
		})
	})
}
