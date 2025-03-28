package client

import (
	"bytes"
	"testing"

	"github.com/buildpacks/lifecycle/auth"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestCommon(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testCommon, spec.Report(report.Terminal{}))
}

func testCommon(t *testing.T, when spec.G, it spec.S) {
	when("#resolveRunImage", func() {
		var (
			subject         *Client
			outBuf          bytes.Buffer
			logger          logging.Logger
			keychain        authn.Keychain
			runImageName    string
			defaultRegistry string
			defaultMirror   string
			gcrRegistry     string
			gcrRunMirror    string
			stackInfo       builder.StackMetadata
			assert          = h.NewAssertionManager(t)
			publish         bool
			err             error
		)

		it.Before(func() {
			logger = logging.NewLogWithWriters(&outBuf, &outBuf)

			keychain, err = auth.DefaultKeychain("pack-test/dummy")
			h.AssertNil(t, err)

			subject, err = NewClient(WithLogger(logger), WithKeychain(keychain))
			assert.Nil(err)

			defaultRegistry = "default.registry.io"
			runImageName = "stack/run"
			defaultMirror = defaultRegistry + "/" + runImageName
			gcrRegistry = "gcr.io"
			gcrRunMirror = gcrRegistry + "/" + runImageName
			stackInfo = builder.StackMetadata{
				RunImage: builder.RunImageMetadata{
					Image: runImageName,
					Mirrors: []string{
						defaultMirror, gcrRunMirror,
					},
				},
			}
		})

		when("passed specific run image", func() {
			it.Before(func() {
				publish = false
			})

			it("selects that run image", func() {
				runImgFlag := "flag/passed-run-image"
				runImageName = subject.resolveRunImage(runImgFlag, defaultRegistry, "", stackInfo.RunImage, nil, publish, image.FetchOptions{Daemon: !publish, PullPolicy: image.PullAlways})
				assert.Equal(runImageName, runImgFlag)
			})
		})

		when("desirable run-image are accessible", func() {
			it.Before(func() {
				publish = true
				mockController := gomock.NewController(t)
				mockFetcher := testmocks.NewMockImageFetcher(mockController)
				mockFetcher.EXPECT().CheckReadAccessValidator(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
				subject, err = NewClient(WithLogger(logger), WithKeychain(keychain), WithFetcher(mockFetcher))
				h.AssertNil(t, err)
			})

			it("defaults to run-image in registry publishing to", func() {
				runImageName = subject.resolveRunImage("", gcrRegistry, defaultRegistry, stackInfo.RunImage, nil, publish, image.FetchOptions{})
				assert.Equal(runImageName, gcrRunMirror)
			})

			it("prefers config defined run image mirror to stack defined run image mirror", func() {
				configMirrors := map[string][]string{
					runImageName: {defaultRegistry + "/unique-run-img"},
				}
				runImageName = subject.resolveRunImage("", defaultRegistry, "", stackInfo.RunImage, configMirrors, publish, image.FetchOptions{})
				assert.NotEqual(runImageName, defaultMirror)
				assert.Equal(runImageName, defaultRegistry+"/unique-run-img")
			})

			it("returns a config mirror if no match to target registry", func() {
				configMirrors := map[string][]string{
					runImageName: {defaultRegistry + "/unique-run-img"},
				}
				runImageName = subject.resolveRunImage("", "test.registry.io", "", stackInfo.RunImage, configMirrors, publish, image.FetchOptions{})
				assert.NotEqual(runImageName, defaultMirror)
				assert.Equal(runImageName, defaultRegistry+"/unique-run-img")
			})
		})

		when("desirable run-images are not accessible", func() {
			it.Before(func() {
				publish = true

				mockController := gomock.NewController(t)
				mockFetcher := testmocks.NewMockImageFetcher(mockController)
				mockFetcher.EXPECT().CheckReadAccessValidator(gcrRunMirror, gomock.Any()).Return(false)
				mockFetcher.EXPECT().CheckReadAccessValidator(stackInfo.RunImage.Image, gomock.Any()).Return(false)
				mockFetcher.EXPECT().CheckReadAccessValidator(defaultMirror, gomock.Any()).Return(true)

				subject, err = NewClient(WithLogger(logger), WithKeychain(keychain), WithFetcher(mockFetcher))
				h.AssertNil(t, err)
			})

			it("selects the first accessible run-image", func() {
				runImageName = subject.resolveRunImage("", gcrRegistry, defaultRegistry, stackInfo.RunImage, nil, publish, image.FetchOptions{})
				assert.Equal(runImageName, defaultMirror)
			})
		})

		when("desirable run-image are empty", func() {
			it.Before(func() {
				publish = false
				stackInfo = builder.StackMetadata{
					RunImage: builder.RunImageMetadata{
						Image: "stack/run-image",
					},
				}
			})

			it("selects the builder run-image", func() {
				// issue: https://github.com/buildpacks/pack/issues/2078
				runImageName = subject.resolveRunImage("", "", "", stackInfo.RunImage, nil, publish, image.FetchOptions{})
				assert.Equal(runImageName, "stack/run-image")
			})
		})
	})
}
