package client_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	cfg "github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/registry"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPullBuildpack(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "PackageBuildpack", testPullBuildpack, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPullBuildpack(t *testing.T, when spec.G, it spec.S) {
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

	when("buildpack has issues", func() {
		it("should fail if not in the registry", func() {
			err := subject.PullBuildpack(context.TODO(), client.PullBuildpackOptions{
				URI:          "invalid/image",
				RegistryName: registry.DefaultRegistryName,
			})
			h.AssertError(t, err, "locating in registry")
		})

		it("should fail if it's a URI type", func() {
			err := subject.PullBuildpack(context.TODO(), client.PullBuildpackOptions{
				URI: "file://some-file",
			})
			h.AssertError(t, err, "unsupported buildpack URI type: 'URILocator'")
		})

		it("should fail if not a valid URI", func() {
			err := subject.PullBuildpack(context.TODO(), client.PullBuildpackOptions{
				URI: "G@Rb*g3_",
			})
			h.AssertError(t, err, "invalid buildpack URI")
		})
	})

	when("pulling from a docker registry", func() {
		it("should fetch the image", func() {
			packageImage := fakes.NewImage("example.com/some/package:1.0.0", "", nil)
			h.AssertNil(t, packageImage.SetLabel("io.buildpacks.buildpackage.metadata", `{}`))
			h.AssertNil(t, packageImage.SetLabel("io.buildpacks.buildpack.layers", `{}`))
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), packageImage.Name(), image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways}).Return(packageImage, nil)

			h.AssertNil(t, subject.PullBuildpack(context.TODO(), client.PullBuildpackOptions{
				URI: "example.com/some/package:1.0.0",
			}))
		})
	})

	when("pulling from a buildpack registry", func() {
		var (
			tmpDir          string
			registryFixture string
			packHome        string
		)

		it.Before(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "registry")
			h.AssertNil(t, err)

			packHome = filepath.Join(tmpDir, ".pack")
			err = os.MkdirAll(packHome, 0755)
			h.AssertNil(t, err)
			os.Setenv("PACK_HOME", packHome)

			registryFixture = h.CreateRegistryFixture(t, tmpDir, filepath.Join("testdata", "registry"))

			packageImage := fakes.NewImage("example.com/some/package@sha256:74eb48882e835d8767f62940d453eb96ed2737de3a16573881dcea7dea769df7", "", nil)
			packageImage.SetLabel("io.buildpacks.buildpackage.metadata", `{}`)
			packageImage.SetLabel("io.buildpacks.buildpack.layers", `{}`)
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), packageImage.Name(), image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways}).Return(packageImage, nil)

			packHome := filepath.Join(tmpDir, "packHome")
			h.AssertNil(t, os.Setenv("PACK_HOME", packHome))
			configPath := filepath.Join(packHome, "config.toml")
			h.AssertNil(t, cfg.Write(cfg.Config{
				Registries: []cfg.Registry{
					{
						Name: "some-registry",
						Type: "github",
						URL:  registryFixture,
					},
				},
			}, configPath))
		})

		it.After(func() {
			os.Unsetenv("PACK_HOME")
			err := os.RemoveAll(tmpDir)
			if runtime.GOOS != "windows" && err != nil && strings.Contains(err.Error(), "The process cannot access the file because it is being used by another process.") {
				h.AssertNil(t, err)
			}
		})

		it("should fetch the image", func() {
			h.AssertNil(t, subject.PullBuildpack(context.TODO(), client.PullBuildpackOptions{
				URI:          "example/foo@1.1.0",
				RegistryName: "some-registry",
			}))
		})
	})
}
