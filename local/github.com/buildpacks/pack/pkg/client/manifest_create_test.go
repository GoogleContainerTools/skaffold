package client

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestCreateManifest(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testCreateManifest, spec.Report(report.Terminal{}))
}

func testCreateManifest(t *testing.T, when spec.G, it spec.S) {
	var (
		mockController   *gomock.Controller
		mockIndexFactory *testmocks.MockIndexFactory
		fakeImageFetcher *ifakes.FakeImageFetcher
		out              bytes.Buffer
		logger           logging.Logger
		subject          *Client
		err              error
		tmpDir           string
	)

	it.Before(func() {
		fakeImageFetcher = ifakes.NewFakeImageFetcher()
		logger = logging.NewLogWithWriters(&out, &out, logging.WithVerbose())
		mockController = gomock.NewController(t)
		mockIndexFactory = testmocks.NewMockIndexFactory(mockController)

		tmpDir, err = os.MkdirTemp("", "add-manifest-test")
		h.AssertNil(t, err)
		os.Setenv("XDG_RUNTIME_DIR", tmpDir)

		subject, err = NewClient(
			WithLogger(logger),
			WithFetcher(fakeImageFetcher),
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

	when("#CreateManifest", func() {
		var indexRepoName string
		when("index doesn't exist", func() {
			var indexLocalPath string

			when("remote manifest is provided", func() {
				it.Before(func() {
					fakeImage := h.NewFakeWithRandomUnderlyingV1Image(t, "pack/image", nil)
					fakeImageFetcher.RemoteImages["index.docker.io/library/busybox:1.36-musl"] = fakeImage
				})

				when("publish is false", func() {
					it.Before(func() {
						// We want to actually create an index, so no need to mock the index factory
						subject, err = NewClient(
							WithLogger(logger),
							WithFetcher(fakeImageFetcher),
							WithExperimental(true),
							WithKeychain(authn.DefaultKeychain),
						)
					})

					when("no errors on save", func() {
						it.Before(func() {
							indexRepoName = h.NewRandomIndexRepoName()
							indexLocalPath = filepath.Join(tmpDir, imgutil.MakeFileSafeName(indexRepoName))
						})

						when("no media type is provided", func() {
							it("creates the index adding the manifest", func() {
								err = subject.CreateManifest(
									context.TODO(),
									CreateManifestOptions{
										IndexRepoName: indexRepoName,
										RepoNames:     []string{"busybox:1.36-musl"},
									},
								)
								h.AssertNil(t, err)
								index := h.ReadIndexManifest(t, indexLocalPath)
								h.AssertEq(t, len(index.Manifests), 1)
								// By default uses OCI media-types
								h.AssertEq(t, index.MediaType, types.OCIImageIndex)
							})
						})

						when("media type is provided", func() {
							it("creates the index adding the manifest", func() {
								err = subject.CreateManifest(
									context.TODO(),
									CreateManifestOptions{
										IndexRepoName: indexRepoName,
										RepoNames:     []string{"busybox:1.36-musl"},
										Format:        types.DockerManifestList,
									},
								)
								h.AssertNil(t, err)
								index := h.ReadIndexManifest(t, indexLocalPath)
								h.AssertEq(t, len(index.Manifests), 1)
								h.AssertEq(t, index.MediaType, types.DockerManifestList)
							})
						})
					})
				})

				when("publish is true", func() {
					var index *h.MockImageIndex

					when("no errors on save", func() {
						it.Before(func() {
							indexRepoName = h.NewRandomIndexRepoName()
							indexLocalPath = filepath.Join(tmpDir, imgutil.MakeFileSafeName(indexRepoName))

							// index stub return to check if push operation was called
							index = h.NewMockImageIndex(t, indexRepoName, 0, 0)

							// We need to mock the index factory to inject a stub index to be pushed.
							mockIndexFactory.EXPECT().Exists(gomock.Eq(indexRepoName)).Return(false)
							mockIndexFactory.EXPECT().CreateIndex(gomock.Eq(indexRepoName), gomock.Any()).Return(index, nil)
						})

						it("creates the index adding the manifest and pushes it to the registry", func() {
							err = subject.CreateManifest(
								context.TODO(),
								CreateManifestOptions{
									IndexRepoName: indexRepoName,
									RepoNames:     []string{"busybox:1.36-musl"},
									Publish:       true,
								},
							)
							h.AssertNil(t, err)

							// index is not saved locally and push it to the registry
							h.AssertPathDoesNotExists(t, indexLocalPath)
							h.AssertTrue(t, index.PushCalled)
							h.AssertTrue(t, index.PurgeOption)
						})
					})
				})
			})

			when("no manifest is provided", func() {
				when("no errors on save", func() {
					it.Before(func() {
						// We want to actually create an index, so no need to mock the index factory
						subject, err = NewClient(
							WithLogger(logger),
							WithFetcher(fakeImageFetcher),
							WithExperimental(true),
							WithKeychain(authn.DefaultKeychain),
						)

						indexRepoName = h.NewRandomIndexRepoName()
						indexLocalPath = filepath.Join(tmpDir, imgutil.MakeFileSafeName(indexRepoName))
					})

					it("creates an empty index with OCI media-type", func() {
						err = subject.CreateManifest(
							context.TODO(),
							CreateManifestOptions{
								IndexRepoName: indexRepoName,
								Format:        types.OCIImageIndex,
							},
						)
						h.AssertNil(t, err)
						index := h.ReadIndexManifest(t, indexLocalPath)
						h.AssertEq(t, len(index.Manifests), 0)
						h.AssertEq(t, index.MediaType, types.OCIImageIndex)
					})

					it("creates an empty index with Docker media-type", func() {
						err = subject.CreateManifest(
							context.TODO(),
							CreateManifestOptions{
								IndexRepoName: indexRepoName,
								Format:        types.DockerManifestList,
							},
						)
						h.AssertNil(t, err)
						index := h.ReadIndexManifest(t, indexLocalPath)
						h.AssertEq(t, len(index.Manifests), 0)
						h.AssertEq(t, index.MediaType, types.DockerManifestList)
					})
				})
			})
		})

		when("index exists", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()

				// mock the index factory to simulate the index exists
				mockIndexFactory.EXPECT().Exists(gomock.Eq(indexRepoName)).AnyTimes().Return(true)
			})

			it("returns an error when index already exists", func() {
				err = subject.CreateManifest(
					context.TODO(),
					CreateManifestOptions{
						IndexRepoName: indexRepoName,
					},
				)
				h.AssertError(t, err, "already exists in local storage; use 'pack manifest remove' to remove it before creating a new manifest list with the same name")
			})
		})
	})
}
