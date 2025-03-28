package client

import (
	"bytes"
	"context"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/internal/registry"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestRegisterBuildpack(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "register_buildpack", testRegisterBuildpack, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testRegisterBuildpack(t *testing.T, when spec.G, it spec.S) {
	when("#RegisterBuildpack", func() {
		var (
			fakeImageFetcher *ifakes.FakeImageFetcher
			fakeAppImage     *fakes.Image
			subject          *Client
			out              bytes.Buffer
		)

		it.Before(func() {
			fakeImageFetcher = ifakes.NewFakeImageFetcher()
			fakeAppImage = fakes.NewImage("buildpack/image", "", &fakeIdentifier{name: "buildpack-image"})

			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.buildpackage.metadata",
				`{"id":"heroku/java-function","version":"1.1.1","stacks":[{"id":"heroku-18"},{"id":"io.buildpacks.stacks.jammy"},{"id":"org.cloudfoundry.stacks.cflinuxfs3"}]}`))
			fakeImageFetcher.RemoteImages["buildpack/image"] = fakeAppImage

			fakeLogger := logging.NewLogWithWriters(&out, &out)
			subject = &Client{
				logger:       fakeLogger,
				imageFetcher: fakeImageFetcher,
			}
		})

		it.After(func() {
			_ = fakeAppImage.Cleanup()
		})

		it("should return error for an invalid image (github)", func() {
			fakeAppImage = fakes.NewImage("invalid/image", "", &fakeIdentifier{name: "buildpack-image"})
			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.buildpackage.metadata", `{}`))

			h.AssertNotNil(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "invalid/image",
					Type:      "github",
					URL:       registry.DefaultRegistryURL,
					Name:      registry.DefaultRegistryName,
				}))
		})

		it("should return error for missing image label (github)", func() {
			fakeAppImage = fakes.NewImage("missinglabel/image", "", &fakeIdentifier{name: "buildpack-image"})
			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.buildpackage.metadata", `{}`))
			fakeImageFetcher.RemoteImages["missinglabel/image"] = fakeAppImage

			h.AssertNotNil(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "missinglabel/image",
					Type:      "github",
					URL:       registry.DefaultRegistryURL,
					Name:      registry.DefaultRegistryName,
				}))
		})

		it("should throw error if missing URL (github)", func() {
			h.AssertError(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "buildpack/image",
					Type:      "github",
					URL:       "",
					Name:      "official",
				}), "missing github URL")
		})

		it("should throw error if missing URL (git)", func() {
			h.AssertError(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "buildpack/image",
					Type:      "git",
					URL:       "",
					Name:      "official",
				}), "invalid url: cannot parse username from url")
		})

		it("should throw error if using malformed URL (git)", func() {
			h.AssertError(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "buildpack/image",
					Type:      "git",
					URL:       "https://github.com//buildpack-registry/",
					Name:      "official",
				}), "invalid url: username is empty")
		})

		it("should return error for an invalid image (git)", func() {
			fakeAppImage = fakes.NewImage("invalid/image", "", &fakeIdentifier{name: "buildpack-image"})
			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.buildpackage.metadata", `{}`))

			h.AssertNotNil(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "invalid/image",
					Type:      "git",
					URL:       registry.DefaultRegistryURL,
					Name:      registry.DefaultRegistryName,
				}))
		})

		it("should return error for missing image label (git)", func() {
			fakeAppImage = fakes.NewImage("missinglabel/image", "", &fakeIdentifier{name: "buildpack-image"})
			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.buildpackage.metadata", `{}`))
			fakeImageFetcher.RemoteImages["missinglabel/image"] = fakeAppImage

			h.AssertNotNil(t, subject.RegisterBuildpack(context.TODO(),
				RegisterBuildpackOptions{
					ImageName: "missinglabel/image",
					Type:      "git",
					URL:       registry.DefaultRegistryURL,
					Name:      registry.DefaultRegistryName,
				}))
		})
	})
}
