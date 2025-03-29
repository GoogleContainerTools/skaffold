package buildpack_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/api"
	"github.com/docker/docker/api/types/system"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuildpackDownloader(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "BuildpackDownloader", testBuildpackDownloader, spec.Report(report.Terminal{}))
}

func testBuildpackDownloader(t *testing.T, when spec.G, it spec.S) {
	var (
		mockController       *gomock.Controller
		mockDownloader       *testmocks.MockBlobDownloader
		mockImageFactory     *testmocks.MockImageFactory
		mockImageFetcher     *testmocks.MockImageFetcher
		mockRegistryResolver *testmocks.MockRegistryResolver
		mockDockerClient     *testmocks.MockCommonAPIClient
		buildpackDownloader  client.BuildpackDownloader
		logger               logging.Logger
		out                  bytes.Buffer
		tmpDir               string
	)

	var createBuildpack = func(descriptor dist.BuildpackDescriptor) string {
		bp, err := ifakes.NewFakeBuildpackBlob(&descriptor, 0644)
		h.AssertNil(t, err)
		url := fmt.Sprintf("https://example.com/bp.%s.tgz", h.RandString(12))
		mockDownloader.EXPECT().Download(gomock.Any(), url).Return(bp, nil).AnyTimes()
		return url
	}

	var createPackage = func(imageName string) *fakes.Image {
		packageImage := fakes.NewImage(imageName, "", nil)
		mockImageFactory.EXPECT().NewImage(packageImage.Name(), false, dist.Target{OS: "linux"}).Return(packageImage, nil)

		pack, err := client.NewClient(
			client.WithLogger(logger),
			client.WithDownloader(mockDownloader),
			client.WithImageFactory(mockImageFactory),
			client.WithFetcher(mockImageFetcher),
			client.WithDockerClient(mockDockerClient),
		)
		h.AssertNil(t, err)

		h.AssertNil(t, pack.PackageBuildpack(context.TODO(), client.PackageBuildpackOptions{
			Name: packageImage.Name(),
			Config: pubbldpkg.Config{
				Platform: dist.Platform{OS: "linux"},
				Buildpack: dist.BuildpackURI{URI: createBuildpack(dist.BuildpackDescriptor{
					WithAPI:    api.MustParse("0.3"),
					WithInfo:   dist.ModuleInfo{ID: "example/foo", Version: "1.1.0"},
					WithStacks: []dist.Stack{{ID: "some.stack.id"}},
				})},
			},
			Publish: true,
		}))

		return packageImage
	}

	it.Before(func() {
		logger = logging.NewLogWithWriters(&out, &out, logging.WithVerbose())
		mockController = gomock.NewController(t)
		mockDownloader = testmocks.NewMockBlobDownloader(mockController)
		mockRegistryResolver = testmocks.NewMockRegistryResolver(mockController)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
		mockImageFactory = testmocks.NewMockImageFactory(mockController)
		mockDockerClient = testmocks.NewMockCommonAPIClient(mockController)
		mockDownloader.EXPECT().Download(gomock.Any(), "https://example.fake/bp-one.tgz").Return(blob.NewBlob(filepath.Join("testdata", "buildpack")), nil).AnyTimes()
		mockDownloader.EXPECT().Download(gomock.Any(), "some/buildpack/dir").Return(blob.NewBlob(filepath.Join("testdata", "buildpack")), nil).AnyTimes()

		buildpackDownloader = buildpack.NewDownloader(logger, mockImageFetcher, mockDownloader, mockRegistryResolver)

		mockDockerClient.EXPECT().Info(context.TODO()).Return(system.Info{OSType: "linux"}, nil).AnyTimes()

		mockRegistryResolver.EXPECT().
			Resolve("some-registry", "urn:cnb:registry:example/foo@1.1.0").
			Return("example.com/some/package@sha256:74eb48882e835d8767f62940d453eb96ed2737de3a16573881dcea7dea769df7", nil).
			AnyTimes()
		mockRegistryResolver.EXPECT().
			Resolve("some-registry", "example/foo@1.1.0").
			Return("example.com/some/package@sha256:74eb48882e835d8767f62940d453eb96ed2737de3a16573881dcea7dea769df7", nil).
			AnyTimes()

		var err error
		tmpDir, err = os.MkdirTemp("", "buildpack-downloader-test")
		h.AssertNil(t, err)
	})

	it.After(func() {
		mockController.Finish()
		h.AssertNil(t, os.RemoveAll(tmpDir))
	})

	when("#Download", func() {
		var (
			packageImage    *fakes.Image
			downloadOptions = buildpack.DownloadOptions{Target: &dist.Target{
				OS: "linux",
			}}
		)

		shouldFetchPackageImageWith := func(demon bool, pull image.PullPolicy, target *dist.Target) {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), packageImage.Name(), image.FetchOptions{
				Daemon:     demon,
				PullPolicy: pull,
				Target:     target,
			}).Return(packageImage, nil)
		}

		when("package image lives in cnb registry", func() {
			it.Before(func() {
				packageImage = createPackage("example.com/some/package@sha256:74eb48882e835d8767f62940d453eb96ed2737de3a16573881dcea7dea769df7")
			})

			when("daemon=true and pull-policy=always", func() {
				it("should pull and use local package image", func() {
					downloadOptions = buildpack.DownloadOptions{
						RegistryName: "some-registry",
						Target:       &dist.Target{OS: "linux", Arch: "amd64"},
						Daemon:       true,
						PullPolicy:   image.PullAlways,
					}

					shouldFetchPackageImageWith(true, image.PullAlways, &dist.Target{OS: "linux", Arch: "amd64"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), "urn:cnb:registry:example/foo@1.1.0", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})

			when("ambigious URI provided", func() {
				it("should find package in registry", func() {
					downloadOptions = buildpack.DownloadOptions{
						RegistryName: "some-registry",
						Target:       &dist.Target{OS: "linux"},
						Daemon:       true,
						PullPolicy:   image.PullAlways,
					}

					shouldFetchPackageImageWith(true, image.PullAlways, &dist.Target{OS: "linux"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), "example/foo@1.1.0", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})
		})

		when("package image lives in docker registry", func() {
			it.Before(func() {
				packageImage = createPackage("docker.io/some/package-" + h.RandString(12))
			})

			prepareFetcherWithMissingPackageImage := func() {
				mockImageFetcher.EXPECT().Fetch(gomock.Any(), packageImage.Name(), gomock.Any()).Return(nil, image.ErrNotFound)
			}

			when("image key is provided", func() {
				it("should succeed", func() {
					packageImage = createPackage("some/package:tag")
					downloadOptions = buildpack.DownloadOptions{
						Daemon:     true,
						PullPolicy: image.PullAlways,
						Target:     &dist.Target{OS: "linux", Arch: "amd64"},
						ImageName:  "some/package:tag",
					}

					shouldFetchPackageImageWith(true, image.PullAlways, &dist.Target{OS: "linux", Arch: "amd64"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), "", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})

			when("daemon=true and pull-policy=always", func() {
				it("should pull and use local package image", func() {
					downloadOptions = buildpack.DownloadOptions{
						Target:     &dist.Target{OS: "linux"},
						ImageName:  packageImage.Name(),
						Daemon:     true,
						PullPolicy: image.PullAlways,
					}

					shouldFetchPackageImageWith(true, image.PullAlways, &dist.Target{OS: "linux"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), "", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})

			when("daemon=false and pull-policy=always", func() {
				it("should use remote package image", func() {
					downloadOptions = buildpack.DownloadOptions{
						Target:     &dist.Target{OS: "linux"},
						ImageName:  packageImage.Name(),
						Daemon:     false,
						PullPolicy: image.PullAlways,
					}

					shouldFetchPackageImageWith(false, image.PullAlways, &dist.Target{OS: "linux"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), "", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})

			when("daemon=false and pull-policy=always", func() {
				it("should use remote package URI", func() {
					downloadOptions = buildpack.DownloadOptions{
						Target:     &dist.Target{OS: "linux"},
						Daemon:     false,
						PullPolicy: image.PullAlways,
					}
					shouldFetchPackageImageWith(false, image.PullAlways, &dist.Target{OS: "linux"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), packageImage.Name(), downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})

			when("publish=true and pull-policy=never", func() {
				it("should push to registry and not pull package image", func() {
					downloadOptions = buildpack.DownloadOptions{
						Target:     &dist.Target{OS: "linux"},
						ImageName:  packageImage.Name(),
						Daemon:     false,
						PullPolicy: image.PullNever,
					}

					shouldFetchPackageImageWith(false, image.PullNever, &dist.Target{OS: "linux"})
					mainBP, _, err := buildpackDownloader.Download(context.TODO(), "", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainBP.Descriptor().Info().ID, "example/foo")
				})
			})

			when("daemon=true pull-policy=never and there is no local package image", func() {
				it("should fail without trying to retrieve package image from registry", func() {
					downloadOptions = buildpack.DownloadOptions{
						Target:     &dist.Target{OS: "linux"},
						ImageName:  packageImage.Name(),
						Daemon:     true,
						PullPolicy: image.PullNever,
					}
					prepareFetcherWithMissingPackageImage()
					_, _, err := buildpackDownloader.Download(context.TODO(), "", downloadOptions)
					h.AssertError(t, err, "not found")
				})
			})
		})

		when("package lives on filesystem", func() {
			it("should successfully retrieve package from absolute path", func() {
				buildpackPath := filepath.Join("testdata", "buildpack")
				buildpackURI, _ := paths.FilePathToURI(buildpackPath, "")
				mockDownloader.EXPECT().Download(gomock.Any(), buildpackURI).Return(blob.NewBlob(buildpackPath), nil).AnyTimes()
				mainBP, _, err := buildpackDownloader.Download(context.TODO(), buildpackURI, downloadOptions)
				h.AssertNil(t, err)
				h.AssertEq(t, mainBP.Descriptor().Info().ID, "bp.one")
			})

			it("should successfully retrieve package from relative path", func() {
				buildpackPath := filepath.Join("testdata", "buildpack")
				buildpackURI, _ := paths.FilePathToURI(buildpackPath, "")
				mockDownloader.EXPECT().Download(gomock.Any(), buildpackURI).Return(blob.NewBlob(buildpackPath), nil).AnyTimes()
				downloadOptions = buildpack.DownloadOptions{
					Target:          &dist.Target{OS: "linux"},
					RelativeBaseDir: "testdata",
				}
				mainBP, _, err := buildpackDownloader.Download(context.TODO(), "buildpack", downloadOptions)
				h.AssertNil(t, err)
				h.AssertEq(t, mainBP.Descriptor().Info().ID, "bp.one")
			})

			when("kind == extension", func() {
				it("succeeds", func() {
					extensionPath := filepath.Join("testdata", "extension")
					extensionURI, _ := paths.FilePathToURI(extensionPath, "")
					mockDownloader.EXPECT().Download(gomock.Any(), extensionURI).Return(blob.NewBlob(extensionPath), nil).AnyTimes()
					downloadOptions = buildpack.DownloadOptions{
						Target:          &dist.Target{OS: "linux"},
						ModuleKind:      "extension",
						RelativeBaseDir: "testdata",
					}
					mainExt, _, err := buildpackDownloader.Download(context.TODO(), "extension", downloadOptions)
					h.AssertNil(t, err)
					h.AssertEq(t, mainExt.Descriptor().Info().ID, "ext.one")
				})
			})

			when("kind == packagedExtension", func() {
				it("succeeds", func() {
					packagedExtensionPath := filepath.Join("testdata", "tree-extension.cnb")
					packagedExtensionURI, _ := paths.FilePathToURI(packagedExtensionPath, "")
					mockDownloader.EXPECT().Download(gomock.Any(), packagedExtensionURI).Return(blob.NewBlob(packagedExtensionPath), nil).AnyTimes()
					downloadOptions = buildpack.DownloadOptions{
						Target:          &dist.Target{OS: "linux"},
						ModuleKind:      "extension",
						RelativeBaseDir: "testdata",
						Daemon:          true,
						PullPolicy:      image.PullAlways,
					}
					mainExt, _, _ := buildpackDownloader.Download(context.TODO(), "tree-extension.cnb", downloadOptions)
					h.AssertEq(t, mainExt.Descriptor().Info().ID, "samples-tree")
				})
			})
		})

		when("package image is not a valid package", func() {
			it("errors", func() {
				notPackageImage := fakes.NewImage("docker.io/not/package", "", nil)

				mockImageFetcher.EXPECT().Fetch(gomock.Any(), notPackageImage.Name(), gomock.Any()).Return(notPackageImage, nil)
				h.AssertNil(t, notPackageImage.SetLabel("io.buildpacks.buildpack.layers", ""))

				downloadOptions.ImageName = notPackageImage.Name()
				_, _, err := buildpackDownloader.Download(context.TODO(), "", downloadOptions)
				h.AssertError(t, err, "extracting buildpacks from 'docker.io/not/package': could not find label 'io.buildpacks.buildpackage.metadata'")
			})
		})

		when("invalid buildpack URI", func() {
			when("buildpack URI is from=builder:fake", func() {
				it("errors", func() {
					_, _, err := buildpackDownloader.Download(context.TODO(), "from=builder:fake", downloadOptions)
					h.AssertError(t, err, "'from=builder:fake' is not a valid identifier")
				})
			})

			when("buildpack URI is from=builder", func() {
				it("errors", func() {
					_, _, err := buildpackDownloader.Download(context.TODO(), "from=builder", downloadOptions)
					h.AssertError(t, err,
						"invalid locator: FromBuilderLocator")
				})
			})

			when("can't resolve buildpack in registry", func() {
				it("errors", func() {
					mockRegistryResolver.EXPECT().
						Resolve("://bad-url", "urn:cnb:registry:fake").
						Return("", errors.New("bad mhkay")).
						AnyTimes()

					downloadOptions.RegistryName = "://bad-url"
					_, _, err := buildpackDownloader.Download(context.TODO(), "urn:cnb:registry:fake", downloadOptions)
					h.AssertError(t, err, "locating in registry")
				})
			})

			when("can't download image from registry", func() {
				it("errors", func() {
					packageImage := fakes.NewImage("example.com/some/package@sha256:74eb48882e835d8767f62940d453eb96ed2737de3a16573881dcea7dea769df7", "", nil)
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), packageImage.Name(), image.FetchOptions{Daemon: false, PullPolicy: image.PullAlways, Target: &dist.Target{OS: "linux"}}).Return(nil, errors.New("failed to pull"))

					downloadOptions.RegistryName = "some-registry"
					_, _, err := buildpackDownloader.Download(context.TODO(), "urn:cnb:registry:example/foo@1.1.0", downloadOptions)
					h.AssertError(t, err,
						"extracting from registry")
				})
			})

			when("buildpack URI is an invalid locator", func() {
				it("errors", func() {
					_, _, err := buildpackDownloader.Download(context.TODO(), "nonsense string here", downloadOptions)
					h.AssertError(t, err,
						"invalid locator: InvalidLocator")
				})
			})
		})
	})
}
