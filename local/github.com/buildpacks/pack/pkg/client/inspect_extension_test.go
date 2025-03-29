package client_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

const extensionMetadataTag = `{
  "id": "some/top-extension",
  "version": "0.0.1",
  "name": "top",
  "homepage": "top-extension-homepage"
}`

const extensionLayersTag = `{
   "some/top-extension":{
      "0.0.1":{
         "api":"0.2",
         "homepage":"top-extension-homepage",
		 "name": "top"
      }
   }
}`

func TestInspectExtension(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "InspectExtension", testInspectExtension, spec.Sequential(), spec.Report(report.Terminal{}))
}
func testInspectExtension(t *testing.T, when spec.G, it spec.S) {
	var (
		subject          *client.Client
		mockImageFetcher *testmocks.MockImageFetcher
		mockController   *gomock.Controller
		out              bytes.Buffer
		extensionImage   *fakes.Image
		expectedInfo     *client.ExtensionInfo
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)

		subject = &client.Client{}
		client.WithLogger(logging.NewLogWithWriters(&out, &out))(subject)
		client.WithFetcher(mockImageFetcher)(subject)

		extensionImage = fakes.NewImage("some/extension", "", nil)
		h.AssertNil(t, extensionImage.SetLabel(dist.ExtensionMetadataLabel, extensionMetadataTag))
		h.AssertNil(t, extensionImage.SetLabel(dist.ExtensionLayersLabel, extensionLayersTag))

		expectedInfo = &client.ExtensionInfo{
			Extension: dist.ModuleInfo{
				ID:       "some/top-extension",
				Version:  "0.0.1",
				Name:     "top",
				Homepage: "top-extension-homepage",
			},
		}
	})

	it.After(func() {
		mockController.Finish()
	})

	when("inspect-extension", func() {
		when("inspecting an image", func() {
			for _, useDaemon := range []bool{true, false} {
				useDaemon := useDaemon
				when(fmt.Sprintf("daemon is %t", useDaemon), func() {
					it.Before(func() {
						expectedInfo.Location = buildpack.PackageLocator
						if useDaemon {
							mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/extension", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(extensionImage, nil)
						} else {
							mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/extension", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(extensionImage, nil)
						}
					})

					it("succeeds", func() {
						inspectOptions := client.InspectExtensionOptions{
							ExtensionName: "docker://some/extension",
							Daemon:        useDaemon,
						}
						info, err := subject.InspectExtension(inspectOptions)
						h.AssertNil(t, err)

						h.AssertEq(t, info, expectedInfo)
					})
				})
			}
		})
	})
	when("failure cases", func() {
		when("invalid extension name", func() {
			it.Before(func() {
				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(nil, errors.Wrapf(image.ErrNotFound, "unable to handle locator"))
			})
			it("returns an error", func() {
				invalidExtensionName := ""
				inspectOptions := client.InspectExtensionOptions{
					ExtensionName: invalidExtensionName,
				}
				_, err := subject.InspectExtension(inspectOptions)

				h.AssertError(t, err, "unable to handle locator")
				h.AssertTrue(t, errors.Is(err, image.ErrNotFound))
			})
		})
		when("extension image", func() {
			when("unable to fetch extension image", func() {
				it.Before(func() {
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "missing/extension", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, errors.Wrapf(image.ErrNotFound, "big bad error"))
				})
				it("returns an ErrNotFound error", func() {
					inspectOptions := client.InspectExtensionOptions{
						ExtensionName: "docker://missing/extension",
						Daemon:        true,
					}
					_, err := subject.InspectExtension(inspectOptions)
					h.AssertTrue(t, errors.Is(err, image.ErrNotFound))
				})
			})

			when("image does not have extension metadata", func() {
				it.Before(func() {
					fakeImage := fakes.NewImage("empty", "", nil)
					h.AssertNil(t, fakeImage.SetLabel(dist.ExtensionLayersLabel, ":::"))
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "missing-metadata/extension", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(fakeImage, nil)
				})
				it("returns an error", func() {
					inspectOptions := client.InspectExtensionOptions{
						ExtensionName: "docker://missing-metadata/extension",
						Daemon:        true,
					}
					_, err := subject.InspectExtension(inspectOptions)

					h.AssertError(t, err, fmt.Sprintf("unable to get image label %s", dist.ExtensionLayersLabel))
					h.AssertFalse(t, errors.Is(err, image.ErrNotFound))
				})
			})
		})
	})
}
