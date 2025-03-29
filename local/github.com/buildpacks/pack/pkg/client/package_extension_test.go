package client_test

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
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPackageExtension(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "PackageExtension", testPackageExtension, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPackageExtension(t *testing.T, when spec.G, it spec.S) {
	var (
		subject          *client.Client
		mockController   *gomock.Controller
		mockDownloader   *testmocks.MockBlobDownloader
		mockImageFactory *testmocks.MockImageFactory
		mockImageFetcher *testmocks.MockImageFetcher
		mockDockerClient *testmocks.MockCommonAPIClient
		out              bytes.Buffer
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockDownloader = testmocks.NewMockBlobDownloader(mockController)
		mockImageFactory = testmocks.NewMockImageFactory(mockController)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
		mockDockerClient = testmocks.NewMockCommonAPIClient(mockController)

		var err error
		subject, err = client.NewClient(
			client.WithLogger(logging.NewLogWithWriters(&out, &out)),
			client.WithDownloader(mockDownloader),
			client.WithImageFactory(mockImageFactory),
			client.WithFetcher(mockImageFetcher),
			client.WithDockerClient(mockDockerClient),
		)
		h.AssertNil(t, err)
	})

	it.After(func() {
		mockController.Finish()
	})

	createExtension := func(descriptor dist.ExtensionDescriptor) string {
		ex, err := ifakes.NewFakeExtensionBlob(&descriptor, 0644)
		h.AssertNil(t, err)
		url := fmt.Sprintf("https://example.com/ex.%s.tgz", h.RandString(12))
		mockDownloader.EXPECT().Download(gomock.Any(), url).Return(ex, nil).AnyTimes()
		return url
	}

	when("extension has issues", func() {
		when("extension has no URI", func() {
			it("should fail", func() {
				err := subject.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
					Name: "Fake-Name",
					Config: pubbldpkg.Config{
						Platform:  dist.Platform{OS: "linux"},
						Extension: dist.BuildpackURI{URI: ""},
					},
					Publish: true,
				})
				h.AssertError(t, err, "extension URI must be provided")
			})
		})

		when("can't download extension", func() {
			it("should fail", func() {
				exURL := fmt.Sprintf("https://example.com/ex.%s.tgz", h.RandString(12))
				mockDownloader.EXPECT().Download(gomock.Any(), exURL).Return(nil, image.ErrNotFound).AnyTimes()

				err := subject.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
					Name: "Fake-Name",
					Config: pubbldpkg.Config{
						Platform:  dist.Platform{OS: "linux"},
						Extension: dist.BuildpackURI{URI: exURL},
					},
					Publish: true,
				})
				h.AssertError(t, err, "downloading buildpack")
			})
		})

		when("extension isn't a valid extension", func() {
			it("should fail", func() {
				fakeBlob := blob.NewBlob(filepath.Join("testdata", "empty-file"))
				exURL := fmt.Sprintf("https://example.com/ex.%s.tgz", h.RandString(12))
				mockDownloader.EXPECT().Download(gomock.Any(), exURL).Return(fakeBlob, nil).AnyTimes()

				err := subject.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
					Name: "Fake-Name",
					Config: pubbldpkg.Config{
						Platform:  dist.Platform{OS: "linux"},
						Extension: dist.BuildpackURI{URI: exURL},
					},
					Publish: true,
				})
				h.AssertError(t, err, "creating extension")
			})
		})
	})

	when("FormatImage", func() {
		when("simple package for both OS formats (experimental only)", func() {
			it("creates package image based on daemon OS", func() {
				for _, daemonOS := range []string{"linux", "windows"} {
					localMockDockerClient := testmocks.NewMockCommonAPIClient(mockController)
					localMockDockerClient.EXPECT().Info(context.TODO()).Return(system.Info{OSType: daemonOS}, nil).AnyTimes()

					packClientWithExperimental, err := client.NewClient(
						client.WithDockerClient(localMockDockerClient),
						client.WithDownloader(mockDownloader),
						client.WithImageFactory(mockImageFactory),
						client.WithExperimental(true),
					)
					h.AssertNil(t, err)

					fakeImage := fakes.NewImage("basic/package-"+h.RandString(12), "", nil)
					mockImageFactory.EXPECT().NewImage(fakeImage.Name(), true, dist.Target{OS: daemonOS}).Return(fakeImage, nil)

					fakeBlob := blob.NewBlob(filepath.Join("testdata", "empty-file"))
					exURL := fmt.Sprintf("https://example.com/ex.%s.tgz", h.RandString(12))
					mockDownloader.EXPECT().Download(gomock.Any(), exURL).Return(fakeBlob, nil).AnyTimes()

					h.AssertNil(t, packClientWithExperimental.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
						Format: client.FormatImage,
						Name:   fakeImage.Name(),
						Config: pubbldpkg.Config{
							Platform: dist.Platform{OS: daemonOS},
							Extension: dist.BuildpackURI{URI: createExtension(dist.ExtensionDescriptor{
								WithAPI:  api.MustParse("0.2"),
								WithInfo: dist.ModuleInfo{ID: "ex.basic", Version: "2.3.4"},
							})},
						},
						PullPolicy: image.PullNever,
					}))
				}
			})

			it("fails without experimental on Windows daemons", func() {
				windowsMockDockerClient := testmocks.NewMockCommonAPIClient(mockController)

				packClientWithoutExperimental, err := client.NewClient(
					client.WithDockerClient(windowsMockDockerClient),
					client.WithExperimental(false),
				)
				h.AssertNil(t, err)

				err = packClientWithoutExperimental.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
					Config: pubbldpkg.Config{
						Platform: dist.Platform{
							OS: "windows",
						},
					},
				})
				h.AssertError(t, err, "Windows extensionpackage support is currently experimental.")
			})

			it("fails for mismatched platform and daemon os", func() {
				windowsMockDockerClient := testmocks.NewMockCommonAPIClient(mockController)
				windowsMockDockerClient.EXPECT().Info(context.TODO()).Return(system.Info{OSType: "windows"}, nil).AnyTimes()

				packClientWithoutExperimental, err := client.NewClient(
					client.WithDockerClient(windowsMockDockerClient),
					client.WithExperimental(false),
				)
				h.AssertNil(t, err)

				err = packClientWithoutExperimental.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
					Config: pubbldpkg.Config{
						Platform: dist.Platform{
							OS: "linux",
						},
					},
				})

				h.AssertError(t, err, "invalid 'platform.os' specified: DOCKER_OS is 'windows'")
			})
		})
	})

	when("FormatFile", func() {
		when("simple package for both OS formats (experimental only)", func() {
			it("creates package image in either OS format", func() {
				tmpDir, err := os.MkdirTemp("", "package-extension")
				h.AssertNil(t, err)
				defer os.Remove(tmpDir)

				for _, imageOS := range []string{"linux", "windows"} {
					localMockDockerClient := testmocks.NewMockCommonAPIClient(mockController)
					localMockDockerClient.EXPECT().Info(context.TODO()).Return(system.Info{OSType: imageOS}, nil).AnyTimes()

					packClientWithExperimental, err := client.NewClient(
						client.WithDockerClient(localMockDockerClient),
						client.WithLogger(logging.NewLogWithWriters(&out, &out)),
						client.WithDownloader(mockDownloader),
						client.WithExperimental(true),
					)
					h.AssertNil(t, err)

					fakeBlob := blob.NewBlob(filepath.Join("testdata", "empty-file"))
					exURL := fmt.Sprintf("https://example.com/ex.%s.tgz", h.RandString(12))
					mockDownloader.EXPECT().Download(gomock.Any(), exURL).Return(fakeBlob, nil).AnyTimes()

					packagePath := filepath.Join(tmpDir, h.RandString(12)+"-test.cnb")
					h.AssertNil(t, packClientWithExperimental.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
						Format: client.FormatFile,
						Name:   packagePath,
						Config: pubbldpkg.Config{
							Platform: dist.Platform{OS: imageOS},
							Extension: dist.BuildpackURI{URI: createExtension(dist.ExtensionDescriptor{
								WithAPI:  api.MustParse("0.2"),
								WithInfo: dist.ModuleInfo{ID: "ex.basic", Version: "2.3.4"},
							})},
						},
						PullPolicy: image.PullNever,
					}))
				}
			})
		})
	})

	when("unknown format is provided", func() {
		it("should error", func() {
			mockDockerClient.EXPECT().Info(context.TODO()).Return(system.Info{OSType: "linux"}, nil).AnyTimes()

			err := subject.PackageExtension(context.TODO(), client.PackageBuildpackOptions{
				Name:   "some-extension",
				Format: "invalid-format",
				Config: pubbldpkg.Config{
					Platform: dist.Platform{OS: "linux"},
					Extension: dist.BuildpackURI{URI: createExtension(dist.ExtensionDescriptor{
						WithAPI:  api.MustParse("0.2"),
						WithInfo: dist.ModuleInfo{ID: "ex.1", Version: "1.2.3"},
					})},
				},
				Publish:    false,
				PullPolicy: image.PullAlways,
			})
			h.AssertError(t, err, "unknown format: 'invalid-format'")
		})
	})
}
