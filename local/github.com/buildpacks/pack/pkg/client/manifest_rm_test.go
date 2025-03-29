package client

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestRemoveManifest(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testRemoveManifest, spec.Report(report.Terminal{}))
}

func testRemoveManifest(t *testing.T, when spec.G, it spec.S) {
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

		tmpDir, err = os.MkdirTemp("", "rm-manifest-test")
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

	when("#RemoveManifest", func() {
		var (
			indexPath     string
			indexRepoName string
		)

		when("index exists", func() {
			var digest name.Digest
			var idx imgutil.ImageIndex

			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				indexPath = filepath.Join(tmpDir, imgutil.MakeFileSafeName(indexRepoName))

				// Initialize the Index with 2 image manifest
				idx, digest = h.RandomCNBIndexAndDigest(t, indexRepoName, 1, 2)
				mockIndexFactory.EXPECT().LoadIndex(gomock.Eq(indexRepoName), gomock.Any()).Return(idx, nil)
			})

			it("should remove local index", func() {
				err = subject.RemoveManifest(indexRepoName, []string{digest.Name()})
				h.AssertNil(t, err)

				// We expect one manifest after removing one of them
				index := h.ReadIndexManifest(t, indexPath)
				h.AssertEq(t, len(index.Manifests), 1)
				h.AssertNotEq(t, index.Manifests[0].Digest.String(), digest.Name())
			})
		})
	})
}
